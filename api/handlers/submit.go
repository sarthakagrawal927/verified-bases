package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Submission is the payload from the /collab creator form.
type Submission struct {
	Title          string `json:"title"`
	OneLiner       string `json:"one_liner"`
	Category       string `json:"category"`
	Price          int    `json:"price"`
	RepoOrDemo     string `json:"repo_or_demo"`
	Stack          string `json:"stack"`
	DoesNotDo      string `json:"does_not_do"`
	Limitations    string `json:"limitations"`
	AIGetsWrong    string `json:"ai_gets_wrong"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	Handle         string `json:"handle"`
	TurnstileToken string `json:"turnstile_token"`
}

// Submit accepts a creator submission, persists it to D1, and pings the
// admin inbox. Reply 202 — review is human.
func Submit(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var s Submission
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeJSON(w, http.StatusBadRequest, `{"ok":false,"error":"invalid_json"}`)
		return
	}
	if clean(s.Title) == "" || clean(s.Email) == "" || clean(s.RepoOrDemo) == "" {
		writeJSON(w, http.StatusUnprocessableEntity, `{"ok":false,"error":"missing_required"}`)
		return
	}

	if ok, code := verifyTurnstile(s.TurnstileToken, clientIP(r)); !ok {
		writeJSON(w, http.StatusForbidden, `{"ok":false,"error":"`+code+`"}`)
		return
	}

	ctx := r.Context()
	st, err := openStore()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, `{"ok":false,"error":"store_unavailable"}`)
		return
	}

	id, err := st.insertSubmission(ctx, s)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, `{"ok":false,"error":"db_write_failed"}`)
		return
	}

	// Notify admin — failure here does not fail the response.
	_, _ = sendEmail(resendEmail{
		To:      []string{adminInbox()},
		Subject: fmt.Sprintf("[bases] new submission: %s", s.Title),
		Text: fmt.Sprintf(
			"New Base submission (id %s)\n\nTitle: %s\nOne-liner: %s\nCategory: %s\nSuggested price: $%d\n\nCreator: %s <%s>%s\nRepo/Demo: %s\nStack: %s\n\n--- does not do ---\n%s\n\n--- known limitations ---\n%s\n\n--- ai gets wrong ---\n%s\n",
			id, s.Title, s.OneLiner, s.Category, s.Price, s.Name, s.Email, handle(s.Handle), s.RepoOrDemo, s.Stack, s.DoesNotDo, s.Limitations, s.AIGetsWrong,
		),
		ReplyTo: s.Email,
	})

	// Confirm to creator.
	_, _ = sendEmail(resendEmail{
		To:      []string{s.Email},
		Subject: "We got your Base submission",
		Text: fmt.Sprintf(
			"Hi %s,\n\nThanks for submitting %q to Verified Software Bases. We review within 7 days and reply on this thread either way.\n\nIf accepted, the next step is polishing the listing copy with you and recording a preview.\n\n— Sarthak",
			pickFirst(s.Name, "there"), s.Title,
		),
	})

	writeJSON(w, http.StatusAccepted,
		fmt.Sprintf(`{"ok":true,"id":"%s","status":"received","review_window_days":7}`, id),
	)
}

func handle(h string) string {
	if h == "" {
		return ""
	}
	return " (" + h + ")"
}

func pickFirst(a, fallback string) string {
	if clean(a) != "" {
		return a
	}
	return fallback
}
