package main


import (
    "context"
    "fmt"
    "net/http"
    "net/http/httputil"
    "net/url"
    "os"

    "github.com/coreos/go-oidc"
    "github.com/gorilla/handlers"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

var (
    ctx        = context.TODO()
    authDomain = getEnv("AUTH_DOMAIN", "https://test.cloudflareaccess.com")
    certsURL   = fmt.Sprintf("%s/cdn-cgi/access/certs", authDomain)

    // policyAUD is your application AUD value
    policyAUD = getEnv("POLICY_AUD", "4714c1358e65fe4b408ad6d432a5f878f08194bdb4752441fd56faefa9b2b6f2")

    listenPort = getEnv("LISTEN_PORT", "8080")
    proxyURL = getEnv("PROXY_URL", "http://default:80")

    config = &oidc.Config{
        ClientID: policyAUD,
    }
    keySet   = oidc.NewRemoteKeySet(ctx, certsURL)
    verifier = oidc.NewVerifier(authDomain, keySet, config)
)

// VerifyToken is a middleware to verify a CF Access token
func VerifyToken(next http.Handler) http.Handler {
    fn := func(w http.ResponseWriter, r *http.Request) {
        headers := r.Header

        // Make sure that the incoming request has our token header
        //  Could also look in the cookies for CF_AUTHORIZATION
        accessJWT := headers.Get("Cf-Access-Jwt-Assertion")
        if accessJWT == "" {
            w.WriteHeader(http.StatusUnauthorized)
            w.Write([]byte("No token on the request"))
            return
        }

        // Verify the access token
        ctx := r.Context()
        _, err := verifier.Verify(ctx, accessJWT)
        if err != nil {
            w.WriteHeader(http.StatusUnauthorized)
            w.Write([]byte(fmt.Sprintf("Invalid token: %s", err.Error())))
            return
        }
        next.ServeHTTP(w, r)
    }
    return http.HandlerFunc(fn)
}

func MainHandler() http.Handler {
    origin, _ := url.Parse(proxyURL)

    director := func(req *http.Request) {
        req.Header.Add("X-Forwarded-Host", req.Host)
        req.Header.Add("X-Origin-Host", origin.Host)
        req.URL.Scheme = "http"
        req.URL.Host = origin.Host
    }

    proxy := &httputil.ReverseProxy{Director: director}

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        proxy.ServeHTTP(w,r)
    })
}

func main() {
    http.Handle("/", VerifyToken(MainHandler()))
    http.ListenAndServe(":" + listenPort, handlers.LoggingHandler(os.Stdout, http.DefaultServeMux))
}
