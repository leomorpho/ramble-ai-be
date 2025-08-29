package webauthn

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type WebAuthnUser struct {
	Record *core.Record
	App    *pocketbase.PocketBase
}

func (u *WebAuthnUser) WebAuthnID() []byte {
	return []byte(u.Record.Id)
}

func (u *WebAuthnUser) WebAuthnName() string {
	return u.Record.GetString("username")
}

func (u *WebAuthnUser) WebAuthnDisplayName() string {
	return u.Record.GetString("name")
}

func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	var credentials []webauthn.Credential
	records, err := u.App.FindAllRecords(
		"passkeys",
		dbx.NewExp("user = {:user}", dbx.Params{"user": u.Record.Id}),
	)
	if err != nil || len(records) == 0 {
		return nil
	}

	for _, record := range records {
		creds := record.GetString("credentials")
		credStruct := webauthn.Credential{}
		if err := json.Unmarshal([]byte(creds), &credStruct); err != nil {
			continue
		}
		credentials = append(credentials, credStruct)
	}
	return credentials
}

var (
	err      error
	webAuthn *webauthn.WebAuthn
)

func findAuthRecordByUsernameOrEmail(app *pocketbase.PocketBase, usernameOrEmail string) (*core.Record, error) {
	// Try to find by email only first
	user, err := app.FindFirstRecordByFilter("users", "email = {:email}", map[string]any{
		"email": usernameOrEmail,
	})

	if err != nil {
		spew.Dump("Error finding user by email:", err)
		// Try by username
		user, err = app.FindFirstRecordByFilter("users", "username = {:username}", map[string]any{
			"username": usernameOrEmail,
		})
		if err != nil {
			spew.Dump("Error finding user by username:", err)
		}
	}

	return user, err
}

func saveCredentials(app *pocketbase.PocketBase, user *core.Record, creds webauthn.Credential) error {
	credID := base64.StdEncoding.EncodeToString(creds.ID) // Parse the credential ID to a string
	passkeyRecord, err := app.FindFirstRecordByFilter("passkeys", "credential_id = {:id}", map[string]any{
		"id": credID,
	})
	if err != nil { // new passkey
		collection, err := app.FindCollectionByNameOrId("passkeys")
		if err != nil {
			return err
		}
		record := core.NewRecord(collection)
		record.Set("user", user.Id)
		record.Set("credential_id", credID)
		record.Set("credentials", creds)
		err = app.Save(record)
		return err
	} else { // update the passkey
		passkeyRecord.Set("credential_id", credID)
		passkeyRecord.Set("credentials", creds)
		err = app.Save(passkeyRecord)
		return err
	}
}

var responses = map[string]any{
	"failed":      "Failed to authenticate",
	"reg_error":   "Failed to register",
	"login_error": "Failed to login",
	"reg_success": "Successfully registered",
	"cred_error":  "Failed to save credentials",
}

func Register(app *pocketbase.PocketBase) {
	origin := app.Settings().Meta.AppURL
	isDevelopment := origin == "" || strings.Contains(origin, "localhost")
	
	if origin == "" {
		origin = "http://localhost:8090" // Default for development
		app.Logger().Warn("WebAuthn: AppURL is not set in config, using default:", origin)
	}
	
	// Extract domain from origin
	var domain string
	if strings.Contains(origin, "://") {
		domain = origin[strings.Index(origin, "//")+2:]
	} else {
		domain = origin
	}
	
	// Remove port from domain for RPID
	if strings.Contains(domain, ":") {
		domain = domain[:strings.Index(domain, ":")]
	}

	var allowedOrigins []string
	
	if isDevelopment {
		// Development: Allow both frontend and backend localhost origins
		allowedOrigins = []string{
			"http://localhost:5174",  // SvelteKit dev server
			"http://localhost:8090",  // PocketBase backend
			"https://localhost:5174", // HTTPS variants
			"https://localhost:8090",
		}
		// Use "localhost" as RPID for cross-port compatibility
		domain = "localhost"
		app.Logger().Info("WebAuthn: Development mode - allowing localhost origins")
	} else {
		// Production: Use the configured origin and its variations
		baseOrigin := strings.TrimSuffix(origin, "/")
		allowedOrigins = []string{baseOrigin}
		
		// Also allow HTTPS variant if origin is HTTP
		if strings.HasPrefix(baseOrigin, "http://") {
			httpsOrigin := "https" + baseOrigin[4:]
			allowedOrigins = append(allowedOrigins, httpsOrigin)
		}
		
		// Also allow HTTP variant if origin is HTTPS (for testing)
		if strings.HasPrefix(baseOrigin, "https://") {
			httpOrigin := "http" + baseOrigin[5:]
			allowedOrigins = append(allowedOrigins, httpOrigin)
		}
		
		app.Logger().Info("WebAuthn: Production mode", "domain", domain, "origins", allowedOrigins)
	}

	wconfig := &webauthn.Config{
		RPDisplayName: app.Settings().Meta.AppName, // Display Name for your site
		RPID:          domain,                      // Domain for WebAuthn (no port)
		RPOrigins:     allowedOrigins,              // The origin URLs allowed for WebAuthn requests
	}

	if webAuthn, err = webauthn.New(wconfig); err != nil {
		fmt.Println(err)
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Start of the registration process
		se.Router.GET("/api/webauthn/registration-options", func(c *core.RequestEvent) error {
			usernameOrEmail := c.Request.URL.Query().Get("usernameOrEmail")
			if usernameOrEmail == "" {
				return c.JSON(400, map[string]string{"error": "usernameOrEmail parameter is required"})
			}

			user, err := findAuthRecordByUsernameOrEmail(app, usernameOrEmail)
			if err != nil {
				return c.JSON(400, responses["failed"])
			}

			waUser := &WebAuthnUser{Record: user, App: app}
			options, session, err := webAuthn.BeginRegistration(waUser)
			if err != nil {
				return c.JSON(500, responses["reg_error"])
			}

			app.Store().Set("webauthn:session:"+user.Id, session)
			return c.JSON(200, options)
		})

		// Finish the registration process
		se.Router.POST("/api/webauthn/register", func(c *core.RequestEvent) error {
			spew.Dump("=== REGISTER ENDPOINT START ===")
			
			info, err := c.RequestInfo() // must get the request info to drain the body text
			if err != nil {
				spew.Dump("Error getting request info:", err)
				return c.JSON(500, map[string]string{"error": "Failed to parse request"})
			}
			
			spew.Dump("Request body:", info.Body)
			
			usernameOrEmailInterface, ok := info.Body["usernameOrEmail"]
			if !ok {
				spew.Dump("usernameOrEmail not found in body")
				return c.JSON(400, map[string]string{"error": "usernameOrEmail is required"})
			}
			
			usernameOrEmail, ok := usernameOrEmailInterface.(string)
			if !ok {
				spew.Dump("usernameOrEmail is not a string")
				return c.JSON(400, map[string]string{"error": "usernameOrEmail must be a string"})
			}
			
			user, err := findAuthRecordByUsernameOrEmail(app, usernameOrEmail)
			if err != nil {
				spew.Dump("User not found:", err)
				return c.JSON(400, responses["failed"])
			}

			sessionInterface := app.Store().Get("webauthn:session:" + user.Id)
			if sessionInterface == nil {
				spew.Dump("Session not found for user:", user.Id)
				return c.JSON(500, map[string]string{"error": "Session not found"})
			}
			
			session, ok := sessionInterface.(*webauthn.SessionData)
			if !ok {
				spew.Dump("Session is not of type *webauthn.SessionData")
				return c.JSON(500, map[string]string{"error": "Invalid session type"})
			}
			
			waUser := &WebAuthnUser{Record: user, App: app}
			creds, err := webAuthn.FinishRegistration(waUser, *session, c.Request)
			if err != nil {
				spew.Dump("FinishRegistration error:", err)
				return c.JSON(500, map[string]string{"error": "Registration failed: " + err.Error()})
			}

			if err := saveCredentials(app, user, *creds); err != nil {
				spew.Dump("saveCredentials error:", err)
				return c.JSON(500, map[string]string{"error": "Failed to save credentials: " + err.Error()})
			}
			
			spew.Dump("=== REGISTER SUCCESS ===")
			return c.JSON(200, responses["reg_success"])
		})

		// Start of the login process
		se.Router.GET("/api/webauthn/login-options", func(c *core.RequestEvent) error {
			usernameOrEmail := c.Request.URL.Query().Get("usernameOrEmail")
			if usernameOrEmail == "" {
				return c.JSON(400, map[string]string{"error": "usernameOrEmail parameter is required"})
			}

			user, err := findAuthRecordByUsernameOrEmail(app, usernameOrEmail)
			if err != nil {
				return c.JSON(400, responses["failed"])
			}

			waUser := &WebAuthnUser{Record: user, App: app}
			options, session, err := webAuthn.BeginLogin(waUser)
			if err != nil {
				return c.JSON(500, responses["login_error"])
			}

			app.Store().Set("webauthn:session:"+user.Id, session)
			return c.JSON(200, options)
		})

		// Finish the login process
		se.Router.POST("/api/webauthn/login", func(c *core.RequestEvent) error {
			info, _ := c.RequestInfo() // must get the request info to drain the body text
			user, err := findAuthRecordByUsernameOrEmail(app, info.Body["usernameOrEmail"].(string))
			if err != nil {
				return c.JSON(400, responses["failed"])
			}

			session := app.Store().Get("webauthn:session:" + user.Id).(*webauthn.SessionData)
			waUser := &WebAuthnUser{Record: user, App: app}
			creds, err := webAuthn.FinishLogin(waUser, *session, c.Request)
			if err != nil {
				return c.JSON(500, responses["login_error"])
			}

			if err := saveCredentials(app, user, *creds); err != nil {
				return c.JSON(500, responses["cred_error"])
			}
			return apis.RecordAuthResponse(c, user, "passkey", nil)
		})
		return se.Next()
	})
}
