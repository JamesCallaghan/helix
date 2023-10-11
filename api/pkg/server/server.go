package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	gocloak "github.com/Nerzal/gocloak/v13"
	"github.com/bacalhau-project/lilysaas/api/pkg/controller"
	"github.com/bacalhau-project/lilysaas/api/pkg/store"
	"github.com/bacalhau-project/lilysaas/api/pkg/system"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
)

type ServerOptions struct {
	URL           string
	Host          string
	Port          int
	KeyCloakURL   string
	KeyCloakToken string
}

type LilysaasAPIServer struct {
	Options    ServerOptions
	Store      store.Store
	Controller *controller.Controller
}

func NewServer(
	options ServerOptions,
	store store.Store,
	controller *controller.Controller,
) (*LilysaasAPIServer, error) {
	if options.URL == "" {
		return nil, fmt.Errorf("server url is required")
	}
	if options.Host == "" {
		return nil, fmt.Errorf("server host is required")
	}
	if options.Port == 0 {
		return nil, fmt.Errorf("server port is required")
	}
	if options.KeyCloakURL == "" {
		return nil, fmt.Errorf("keycloak url is required")
	}
	if options.KeyCloakToken == "" {
		return nil, fmt.Errorf("keycloak token is required")
	}

	return &LilysaasAPIServer{
		Options:    options,
		Store:      store,
		Controller: controller,
	}, nil
}

type keycloak struct {
	gocloak      *gocloak.GoCloak // keycloak client
	externalUrl  string           // the URL of the keycloak server
	clientId     string           // clientId specified in Keycloak
	clientSecret string           // client secret specified in Keycloak
	realm        string           // realm specified in Keycloak
}

const CLIENT_ID = "api"
const REALM = "lilypad"

func (apiServer *LilysaasAPIServer) newKeycloak() *keycloak {
	externalUrl := apiServer.Options.KeyCloakURL
	keycloak := &keycloak{
		gocloak:      gocloak.NewClient(externalUrl),
		externalUrl:  externalUrl,
		clientId:     CLIENT_ID,
		clientSecret: apiServer.Options.KeyCloakToken,
		realm:        REALM,
	}
	return keycloak
}

type keyCloakMiddleware struct {
	keycloak *keycloak
	options  ServerOptions
}

func newMiddleware(keycloak *keycloak, options ServerOptions) *keyCloakMiddleware {
	return &keyCloakMiddleware{keycloak: keycloak, options: options}
}

func extractBearerToken(token string) string {
	return strings.Replace(token, "Bearer ", "", 1)
}

func (auth *keyCloakMiddleware) userFromRequest(r *http.Request) (*jwt.Token, error) {
	// try to extract Authorization parameter from the HTTP header
	token := r.Header.Get("Authorization")
	if token == "" {
		return nil, fmt.Errorf("authorization header missing")
	}

	// extract Bearer token
	token = extractBearerToken(token)
	if token == "" {
		return nil, fmt.Errorf("bearer token missing")
	}

	result, err := auth.keycloak.gocloak.RetrospectToken(r.Context(), token, CLIENT_ID, auth.options.KeyCloakToken, REALM)
	if err != nil {
		return nil, fmt.Errorf("RetrospectToken: invalid or malformed token: %s", err.Error())
	}

	j, _, err := auth.keycloak.gocloak.DecodeAccessToken(r.Context(), token, REALM)
	if err != nil {
		return nil, fmt.Errorf("DecodeAccessToken: invalid or malformed token: %s", err.Error())
	}

	// check if the token isn't expired and valid
	if !*result.Active {
		return nil, fmt.Errorf("invalid or expired token")
	}

	return j, nil
}

func getUserIdFromJWT(tok *jwt.Token) string {
	mc := tok.Claims.(jwt.MapClaims)
	uid := mc["sub"].(string)
	return uid
}

func setRequestUser(ctx context.Context, userid string) context.Context {
	return context.WithValue(ctx, "userid", userid)
}

func getRequestUser(req *http.Request) string {
	// return the "userid" value from the request context
	return req.Context().Value("userid").(string)
}

func (auth *keyCloakMiddleware) verifyToken(next http.Handler) http.Handler {

	f := func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.userFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		r = r.WithContext(setRequestUser(r.Context(), getUserIdFromJWT(token)))
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(f)
}

func (apiServer *LilysaasAPIServer) ListenAndServe(ctx context.Context, cm *system.CleanupManager) error {
	router := mux.NewRouter()
	router.Use(apiServer.corsMiddleware)

	subrouter := router.PathPrefix("/api/v1").Subrouter()

	// add one more subrouter for the authenticated service methods
	authRouter := subrouter.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return true
	}).Subrouter()

	kc := apiServer.newKeycloak()
	mdw := newMiddleware(kc, apiServer.Options)
	authRouter.Use(mdw.verifyToken)

	subrouter.HandleFunc("/modules", wrapper(apiServer.getModules)).Methods("GET")

	authRouter.HandleFunc("/status", wrapper(apiServer.status)).Methods("GET")
	authRouter.HandleFunc("/jobs", wrapper(apiServer.getJobs)).Methods("GET")
	authRouter.HandleFunc("/jobs", wrapper(apiServer.createJob)).Methods("POST")

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", apiServer.Options.Host, apiServer.Options.Port),
		WriteTimeout:      time.Minute * 15,
		ReadTimeout:       time.Minute * 15,
		ReadHeaderTimeout: time.Minute * 15,
		IdleTimeout:       time.Minute * 60,
		Handler:           router,
	}
	return srv.ListenAndServe()
}
