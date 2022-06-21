package handler

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type mockSESv2CreateContactAPI struct {
	*testing.T
	apiError error
}

func (m *mockSESv2CreateContactAPI) CreateContact(
	ctx context.Context,
	params *sesv2.CreateContactInput,
	optFns ...func(*sesv2.Options),
) (*sesv2.CreateContactOutput, error) {
	if params == nil {
		m.Error("got nil for params")
	}
	if params.EmailAddress == nil || len(*params.EmailAddress) == 0 {
		m.Error("got empty or nil string for params.EmailAddress")
	}
	if params.ContactListName == nil || len(*params.ContactListName) == 0 {
		m.Error("got empty or nil string for params.ContactListName")
	}
	if len(params.TopicPreferences) == 0 {
		m.Error("got empty list for params.TopicPreferences")
	} else {
		for i, tp := range params.TopicPreferences {
			if tp.TopicName == nil || len(*tp.TopicName) == 0 {
				m.Errorf("got empty or nil string for topic pref %d", i)
			}
			if tp.SubscriptionStatus != types.SubscriptionStatusOptIn {
				m.Errorf("incorrect subscription status for %s: got OPT_OUT; expected OPT_IN", *tp.TopicName)
			}
		}
	}
	return &sesv2.CreateContactOutput{}, m.apiError
}

func TestHandler(t *testing.T) {
	testCases := []struct {
		email          string
		clName         string
		apiError       error
		expectedStatus int
	}{
		{"email1@example.com", "contacts", nil, 200},
		{"", "contacts", nil, 400},
		{"email2@example.com", "contacts", errors.New("error invalid"), 500},
		{"email3@example.com", "contacts", fmt.Errorf("error: %w", &types.AlreadyExistsException{}), 409},
	}

	for _, tc := range testCases {
		var expectErrorDesc string
		if tc.apiError != nil {
			expectErrorDesc = " with error"
		} else {
			expectErrorDesc = ""
		}
		t.Run(fmt.Sprintf("error status %d with email %s%s", tc.expectedStatus, tc.email, expectErrorDesc), func(t *testing.T) {
			m := &mockSESv2CreateContactAPI{t, tc.apiError}
			h := New(m, tc.clName)
			out, err := h.Subscribe(events.APIGatewayV2HTTPRequest{
				QueryStringParameters: map[string]string{
					"email": tc.email,
				},
			})

			// Should pass through SDK errors
			if err != nil && !errors.Is(err, tc.apiError) {
				t.Errorf("unexpected error value: got %v; expected %v", err, tc.apiError)
			}

			if out.StatusCode != tc.expectedStatus {
				t.Errorf("unexpected StatusCode value: got %d; expected %d", out.StatusCode, tc.expectedStatus)
			}
		})
	}
}
