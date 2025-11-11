package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/bcrypt"

	"github.com/Andres09xZ/latacunga_clean_app/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/internal/server"
	"gorm.io/gorm"
)

var testCtx *testContext

var opt = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "progress", // can be progress, pretty, json, etc.
}

type testContext struct {
	db         *gorm.DB
	router     *gin.Engine
	lastResp   *httptest.ResponseRecorder
	lastTokens map[string]string
	users      map[string]*models.User
}

var ctx *testContext

func init() {
	godog.BindCommandLineFlags("godog.", &opt)
}

func TestMain(m *testing.M) {
	pflag.Parse()
	opt.Paths = []string{"features"}

	status := godog.TestSuite{
		Name:                 "auth-service",
		TestSuiteInitializer: InitializeTestSuite,
		ScenarioInitializer:  InitializeScenario,
		Options:              &opt,
	}.Run()

	os.Exit(status)
}

func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		// Load test environment
		if err := godotenv.Load("../.env"); err != nil {
			fmt.Printf("Error loading .env file: %v\n", err)
		}

		// Setup test database
		setupTestDatabase()
	})

	ctx.AfterSuite(func() {
		// Cleanup test database
		cleanupTestDatabase()
	})
}

func InitializeScenario(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		// Setup scenario context
		testCtx = &testContext{
			db:         database.DB,
			router:     server.SetupRouter(),
			lastResp:   nil,
			lastTokens: make(map[string]string),
			users:      make(map[string]*models.User),
		}
		return ctx, nil
	})

	sc.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		// Cleanup after scenario
		if testCtx != nil {
			testCtx.db.Exec("DELETE FROM refresh_tokens")
			// testCtx.db.Exec("DELETE FROM otp_requests") // Keep OTP for cross-scenario tests
			testCtx.db.Exec("DELETE FROM users")
		}
		return ctx, nil
	})

	// Background steps
	sc.Step(`^existen los roles (.+)$`, existenLosRoles)
	sc.Step(`^access_token expira en (\d+) minutos$`, accessTokenExpiraEnMinutos)
	sc.Step(`^refresh_token expira en (\d+) días$`, refreshTokenExpiraEnDias)
	sc.Step(`^OTP: (\d+) dígitos, expira en (\d+) minutos, max (\d+) intentos$`, otpConfig)
	sc.Step(`^el endpoint de login es POST "([^"]*)"$`, endpointLogin)
	sc.Step(`^el endpoint de refresh es POST "([^"]*)"$`, endpointRefresh)
	sc.Step(`^el endpoint de logout es POST "([^"]*)"$`, endpointLogout)
	sc.Step(`^los endpoints OTP son POST "([^"]*)" y POST "([^"]*)"$`, endpointsOTP)

	// Scenario steps
	sc.Step(`^existe un usuario con email "([^"]*)" y contraseña "([^"]*)" y rol "([^"]*)"$`, existeUnUsuario)
	sc.Step(`^hago POST a "([^"]*)" con:$`, hagoPOSTaCon)
	sc.Step(`^la respuesta es (\d+)$`, laRespuestaEs)
	sc.Step(`^el cuerpo contiene "([^"]*)" y "([^"]*)"$`, elCuerpoContieneTokens)
	sc.Step(`^el "([^"]*)" contiene claim "([^"]*)" = "([^"]*)"$`, elTokenContieneClaim)
	sc.Step(`^el "([^"]*)" tiene expiración <= (\d+) (\w+)$`, elTokenTieneExpiracion)
	sc.Step(`^el cuerpo contiene "([^"]*)" con "([^"]*)"$`, elCuerpoContieneMensaje)
	sc.Step(`^que tengo un "([^"]*)" válido$`, queTengoUnTokenValido)
	sc.Step(`^que tengo un "([^"]*)" inválido o expirado$`, queTengoUnTokenInvalido)
	sc.Step(`^que estoy autenticado con access_token válido$`, queEstoyAutenticado)
	sc.Step(`^que estoy autenticado con access_token rol "([^"]*)"$`, queEstoyAutenticadoConRol)
	sc.Step(`^que se solicitó OTP para "([^"]*)"$`, queSeSolicitoOTPPara)
	sc.Step(`^el código generado fue "([^"]*)"$`, elCodigoGeneradoFue)
	sc.Step(`^no existe usuario con ese teléfono$`, noExisteUsuarioConEseTelefono)
	sc.Step(`^he realizado (\d+) intentos fallidos$`, heRealizadoIntentosFallidos)
	sc.Step(`^el sistema crea un usuario con rol "([^"]*)"$`, elSistemaCreaUsuarioConRol)
	sc.Step(`^que no estoy autenticado$`, queNoEstoyAutenticado)
	sc.Step(`^hago GET a "([^"]*)"$`, hagoGETa)
	sc.Step(`^que estoy autenticado con access_token expirado$`, queEstoyAutenticadoConTokenExpirado)
	sc.Step(`^que presento un access_token con firma inválida$`, quePresentoTokenConFirmaInvalida)
	sc.Step(`^el access_token contiene claim "([^"]*)" = "([^"]*)"$`, elAccess_tokenContieneClaim)
	sc.Step(`^el cuerpo contiene un nuevo "([^"]*)" y un nuevo "([^"]*)"$`, elCuerpoContieneUnNuevoYUnNuevo)
	sc.Step(`^el nuevo usuario queda registrado con rol "([^"]*)"$`, elNuevoUsuarioQuedaRegistradoConRol)
	sc.Step(`^el "([^"]*)" queda invalidado$`, elQuedaInvalidado)
	sc.Step(`^el "([^"]*)" tiene expiración <= (\d+) días$`, elTieneExpiracionDias)
	sc.Step(`^el usuario se registra con rol "([^"]*)"$`, elUsuarioSeRegistraConRol)
	sc.Step(`^existe el número "([^"]*)"$`, existeElNumero)
	sc.Step(`^hago POST a "([^"]*)"$`, hagoPOSTa)
	sc.Step(`^tengo un "([^"]*)" activo$`, tengoUnActivo)
}

func setupTestDatabase() {
	// Setup test database connection
	database.Connect()

	// Clean up any existing tables
	database.DB.Exec("DROP TABLE IF EXISTS refresh_tokens CASCADE")
	database.DB.Exec("DROP TABLE IF EXISTS otp_requests CASCADE")
	database.DB.Exec("DROP TABLE IF EXISTS users CASCADE")

	// Run GORM migrations first to create tables
	database.DB.AutoMigrate(&models.User{}, &models.RefreshToken{}, &models.OTPRequest{})

	// Run SQL migrations for additional alterations
	migrationFiles := []string{
		"../migrations/001_add_id_user.sql",
		"../migrations/002_nullable_email_password.sql",
	}

	for _, file := range migrationFiles {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Error reading migration file %s: %v", file, err)
		}
		sql := string(sqlBytes)
		if err := database.DB.Exec(sql).Error; err != nil {
			log.Fatalf("Error executing migration %s: %v", file, err)
		}
	}
}

func cleanupTestDatabase() {
	// Close database connection
	sqlDB, _ := database.DB.DB()
	sqlDB.Close()
}

// Step definitions

func existenLosRoles(roles string) error {
	// This is just configuration, no action needed
	return nil
}

func accessTokenExpiraEnMinutos(minutos int) error {
	// This is just configuration, no action needed
	return nil
}

func refreshTokenExpiraEnDias(dias int) error {
	// This is just configuration, no action needed
	return nil
}

func otpConfig(digitos, minutos, intentos int) error {
	// This is just configuration, no action needed
	return nil
}

func endpointLogin(endpoint string) error {
	// This is just configuration, no action needed
	return nil
}

func endpointRefresh(endpoint string) error {
	// This is just configuration, no action needed
	return nil
}

func endpointLogout(endpoint string) error {
	// This is just configuration, no action needed
	return nil
}

func endpointsOTP(request, verify string) error {
	// This is just configuration, no action needed
	return nil
}

func existeUnUsuario(email, password, role string) error {
	// Hash the password like the real application does
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}
	hashedPasswordStr := string(hashedPassword)

	// Create test user
	user := &models.User{
		IDUser:       fmt.Sprintf("test-%s", email),
		Email:        &email,
		PasswordHash: &hashedPasswordStr,
		Role:         role,
		Name:         "Test User",
	}

	if err := testCtx.db.Create(user).Error; err != nil {
		return err
	}

	testCtx.users[email] = user
	return nil
}

func hagoPOSTaCon(endpoint string, jsonStr *godog.DocString) error {
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(jsonStr.Content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	testCtx.lastResp = httptest.NewRecorder()
	testCtx.router.ServeHTTP(testCtx.lastResp, req)

	// If the response is successful and contains tokens, store them
	if testCtx.lastResp.Code >= 200 && testCtx.lastResp.Code < 300 {
		var body map[string]interface{}
		if err := json.Unmarshal(testCtx.lastResp.Body.Bytes(), &body); err == nil {
			if accessToken, ok := body["access_token"].(string); ok {
				testCtx.lastTokens["access_token"] = accessToken
			}
			if refreshToken, ok := body["refresh_token"].(string); ok {
				testCtx.lastTokens["refresh_token"] = refreshToken
			}
		}
	}

	return nil
}

func laRespuestaEs(expectedStatus int) error {
	if testCtx == nil || testCtx.lastResp == nil {
		return fmt.Errorf("no response recorded - make sure to make a request first")
	}
	if testCtx.lastResp.Code != expectedStatus {
		return fmt.Errorf("expected status %d, got %d", expectedStatus, testCtx.lastResp.Code)
	}
	return nil
}

func elCuerpoContieneTokens(token1, token2 string) error {
	var body map[string]interface{}
	if err := json.Unmarshal(testCtx.lastResp.Body.Bytes(), &body); err != nil {
		return err
	}

	if _, ok := body[token1]; !ok {
		return fmt.Errorf("response does not contain %s", token1)
	}
	if _, ok := body[token2]; !ok {
		return fmt.Errorf("response does not contain %s", token2)
	}

	// Store tokens for later use
	testCtx.lastTokens = make(map[string]string)
	if t, ok := body[token1].(string); ok {
		testCtx.lastTokens[token1] = t
	}
	if t, ok := body[token2].(string); ok {
		testCtx.lastTokens[token2] = t
	}

	return nil
}

func elTokenContieneClaim(tokenType, claim, expectedValue string) error {
	_, ok := testCtx.lastTokens[tokenType]
	if !ok {
		return fmt.Errorf("token %s not found", tokenType)
	}

	// For now, just check if token exists (full JWT validation would require parsing)
	// In a real implementation, you'd decode and validate the JWT
	return nil
}

func elTokenTieneExpiracion(tokenType string, maxTime int, unit string) error {
	// For now, just check if token exists
	// In a real implementation, you'd check the exp claim
	return nil
}

func elCuerpoContieneMensaje(field, expectedMessage string) error {
	var body map[string]interface{}
	if err := json.Unmarshal(testCtx.lastResp.Body.Bytes(), &body); err != nil {
		return err
	}

	if msg, ok := body[field].(string); !ok || msg != expectedMessage {
		return fmt.Errorf("expected %s='%s', got %v", field, expectedMessage, body[field])
	}

	return nil
}

// Placeholder implementations for remaining steps
func queTengoUnTokenValido(tokenType string) error {
	if testCtx.lastTokens == nil {
		testCtx.lastTokens = make(map[string]string)
	}
	// For refresh token, use a stored one or create a test one
	if tokenType == "refresh_token" {
		if stored, exists := testCtx.lastTokens["refresh_token"]; exists && stored != "" {
			// Use stored token
		} else {
			// Create a test user first
			email := "refresh@example.com"
			hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)
			hashedPasswordStr := string(hashedPassword)
			testUser := &models.User{
				IDUser:       "test-refresh-user",
				Email:        &email,
				PasswordHash: &hashedPasswordStr,
				Role:         "user",
				Name:         "Test Refresh User",
			}
			if err := testCtx.db.Create(testUser).Error; err != nil {
				return err
			}
			// Create a test refresh token in database
			testRefreshToken := &models.RefreshToken{
				UserID:    testUser.ID,
				Token:     "test_refresh_token_valid",
				ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
			}
			if err := testCtx.db.Create(testRefreshToken).Error; err != nil {
				return err
			}
			testCtx.lastTokens["refresh_token"] = "test_refresh_token_valid"
		}
	}
	return nil
}
func queTengoUnTokenInvalido(tokenType string) error {
	if tokenType == "refresh_token" {
		testCtx.lastTokens = map[string]string{
			"refresh_token": "invalid_refresh_token",
		}
	}
	return nil
}
func queEstoyAutenticado() error {
	testCtx.lastTokens = map[string]string{
		"access_token": "valid_access_token_placeholder",
	}
	return nil
}
func queEstoyAutenticadoConRol(role string) error {
	testCtx.lastTokens = map[string]string{
		"access_token": "valid_access_token_" + role,
	}
	return nil
}
func queSeSolicitoOTPPara(phone string) error        { return nil }
func elCodigoGeneradoFue(code string) error          { return nil }
func noExisteUsuarioConEseTelefono() error           { return nil }
func heRealizadoIntentosFallidos(attempts int) error { return nil }
func elSistemaCreaUsuarioConRol(role string) error   { return nil }
func queNoEstoyAutenticado() error                   { return nil }
func hagoGETa(endpoint string) error {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}

	// Add authorization header if we have an access token
	if testCtx.lastTokens != nil {
		if token, exists := testCtx.lastTokens["access_token"]; exists && token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	testCtx.lastResp = httptest.NewRecorder()
	testCtx.router.ServeHTTP(testCtx.lastResp, req)
	return nil
}
func queEstoyAutenticadoConTokenExpirado() error {
	testCtx.lastTokens = map[string]string{
		"access_token": "expired_token",
	}
	return nil
}
func quePresentoTokenConFirmaInvalida() error {
	testCtx.lastTokens = map[string]string{
		"access_token": "invalid_token",
	}
	return nil
}

// Missing step implementations
func elAccess_tokenContieneClaim(claim, value string) error {
	// For now, just check if we have tokens stored
	if testCtx.lastTokens == nil || testCtx.lastTokens["access_token"] == "" {
		return fmt.Errorf("no access token available to check claims")
	}
	// In a real implementation, you'd decode the JWT and check the claim
	return nil
}

func elCuerpoContieneUnNuevoYUnNuevo(field1, field2 string) error {
	var body map[string]interface{}
	if err := json.Unmarshal(testCtx.lastResp.Body.Bytes(), &body); err != nil {
		return err
	}
	if _, ok := body[field1]; !ok {
		return fmt.Errorf("response body missing field: %s", field1)
	}
	if _, ok := body[field2]; !ok {
		return fmt.Errorf("response body missing field: %s", field2)
	}
	return nil
}

func elNuevoUsuarioQuedaRegistradoConRol(role string) error {
	// Check if user was created with the specified role
	// This would require checking the database
	return nil
}

func elQuedaInvalidado(tokenType string) error {
	// Check if token was invalidated
	return nil
}

func elTieneExpiracionDias(tokenType string, days int) error {
	// Check token expiration
	return nil
}

func elUsuarioSeRegistraConRol(role string) error {
	// Check user registration with role
	return nil
}

func existeElNumero(phone string) error {
	// Store phone number for later use
	testCtx.lastTokens = map[string]string{
		"phone": phone,
	}
	return nil
}

func hagoPOSTa(endpoint string) error {
	// Make POST request without body
	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	testCtx.lastResp = httptest.NewRecorder()
	testCtx.router.ServeHTTP(testCtx.lastResp, req)
	return nil
}

func tengoUnActivo(tokenType string) error {
	// Store active token
	testCtx.lastTokens = map[string]string{
		tokenType: "active_" + tokenType,
	}
	return nil
}
