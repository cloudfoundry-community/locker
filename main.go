package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/voxelbrain/goptions"
)

var Version string

func main() {
	var options struct {
		Help    bool `goptions:"-h, --help"`
		Version bool `goptions:"-v, --version"`
	}

	goptions.ParseAndFail(&options)
	if options.Help {
		goptions.PrintHelp()
		os.Exit(0)
	}
	if options.Version {
		if Version != "" {
			fmt.Fprintf(os.Stderr, "locker v%s\n", Version)
		} else {
			fmt.Fprintf(os.Stderr, "locker (development build)\n")
		}
		os.Exit(0)
	}

	lockChan := make(chan LockRequest)
	lockConfig := os.Getenv("LOCKER_CONFIG")
	if lockConfig == "" {
		fmt.Fprintf(os.Stderr, "No LOCKER_CONFIG file specified. Bailing out.\n")
		os.Exit(1)
	}
	_, err := os.Stat(lockConfig)
	if os.IsNotExist(err) {
		err := ioutil.WriteFile(lockConfig, []byte("{}"), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create LOCKER_CONFIG file (%s): %s\n", lockConfig, err)
			fmt.Fprintf(os.Stderr, "Bailing out.\n")
			os.Exit(1)
		}
	}

	authUser := os.Getenv("AUTH_USER")
	authPass := os.Getenv("AUTH_PASS")
	if authUser != "" && authPass == "" {
		fmt.Fprintf(os.Stderr, "AUTH_USER specified, but no AUTH_PASS was provided. Bailing out.\n")
		os.Exit(1)
	}
	if authUser == "" && authPass != "" {
		fmt.Fprintf(os.Stderr, "AUTH_PASS specified, but no AUTH_USER was provided. Bailing out.\n")
		os.Exit(1)
	}

	certFile := os.Getenv("SSL_CERT")
	keyFile := os.Getenv("SSL_KEY")
	if certFile == "" && keyFile != "" {
		fmt.Fprintf(os.Stderr, "SSL_KEY specified, but no SSL_CERT was provided. Bailing out.\n")
		os.Exit(1)
	}
	if certFile != "" && keyFile == "" {
		fmt.Fprintf(os.Stderr, "SSL_CERT specified, but no SSL_KEY was provided. Bailing out.\n")
		os.Exit(1)
	}

	go lockServer(lockChan, lockConfig)

	router := gin.Default()
	var routes *gin.RouterGroup
	if authUser != "" && authPass != "" {
		routes = router.Group("/", AuthHandler(authUser, authPass))
	} else {
		routes = router.Group("/", func(c *gin.Context) {})
	}
	routes.GET("/locks", func(c *gin.Context) {
		rc := make(chan LockResponse)

		lr := LockRequest{
			Command:  ListOp,
			Response: rc,
		}

		lockChan <- lr

		response := <-rc
		if response.Error != nil {
			fmt.Fprintf(os.Stderr, "Error listing locks: %s\n", response.Error)
			c.Render(500, render.JSON{Data: map[string]string{"error": response.Error.Error()}})
			return
		}

		c.Render(200, render.JSON{Data: response.Message})
	})
	routes.PUT("/lock/:pool", func(c *gin.Context) {
		rc := make(chan LockResponse)

		pool := c.Param("pool")
		input := LockInput{}
		c.BindJSON(&input)
		if input.Key == "" {
			c.Render(400, render.JSON{Data: map[string]string{"error": "No key specified in request body"}})
			return
		}
		if input.Requestor == "" {
			input.Requestor = input.Key
		}

		lr := LockRequest{
			Command:  LockOp,
			Pool:     pool,
			Lock:     input,
			Response: rc,
		}
		lockChan <- lr
		res := <-rc
		if res.Error != nil {
			fmt.Fprintf(os.Stderr, "Error locking %s with %s: %s\n", pool, input.Key, res.Error)
			c.Render(423, render.JSON{Data: map[string]string{"error": res.Error.Error()}})
			return
		}

		if res.Status == Locked {
			c.Render(200, render.JSON{Data: res.Message})
		} else {
			fmt.Fprintf(os.Stderr, "Unexpected response to a lock request: %#v\n", res)
			c.Render(500, render.JSON{Data: map[string]string{"error": "Unknown response from lock request"}})
		}
	})
	routes.DELETE("/lock/:pool", func(c *gin.Context) {
		rc := make(chan LockResponse)
		pool := c.Param("pool")
		input := LockInput{}
		c.BindJSON(&input)
		if input.Key == "" {
			c.Render(400, render.JSON{Data: map[string]string{"error": "No lock specified in request body"}})
			return
		}
		if input.Requestor == "" {
			input.Requestor = input.Key
		}

		lr := LockRequest{
			Command:  UnlockOp,
			Pool:     pool,
			Lock:     input,
			Response: rc,
		}

		lockChan <- lr
		res := <-rc
		if res.Error != nil {
			fmt.Fprintf(os.Stderr, "Error unlocking %s: %s\n", pool, res.Error)
			c.Render(423, render.JSON{Data: map[string]string{"error": res.Error.Error()}})
			return
		}

		if res.Status != Unlocked {
			fmt.Fprintf(os.Stderr, "Unexpected response to unlock request: %#v\n", res)
			c.Render(500, render.JSON{Data: map[string]string{"error": "Unknown response from unlock request"}})
			return
		}

		c.Render(200, render.JSON{Data: res.Message})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	if certFile == "" && keyFile == "" {
		err := http.ListenAndServe(":"+port, router)
		fmt.Fprintf(os.Stderr, "Error running webserver: %s\n", err)
		os.Exit(1)
	} else if certFile != "" && keyFile != "" {
		err := http.ListenAndServeTLS(":"+port, certFile, keyFile, router)
		fmt.Fprintf(os.Stderr, "Error running webserver: %s\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Unable to choose TLS or non-TLS web server. Bailing out\n")
	os.Exit(0)
}

func AuthHandler(user, password string) gin.HandlerFunc {
	realm := "Basic realm=\"Authorization Required\""
	return func(c *gin.Context) {
		hdr := c.Request.Header.Get("Authorization")
		fields := strings.SplitN(hdr, " ", 2)
		if strings.ToLower(fields[0]) != "basic" {
			fmt.Fprintf(os.Stderr, "Got illegal authentication request type: %s\n", fields[0])
			c.Header("WWW-Authenticate", realm)
			c.Render(401, render.JSON{Data: map[string]string{"error": "Please Authenticate"}})
			c.Abort()
			return
		}

		auth, err := base64.StdEncoding.DecodeString(fields[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to decode authorization header '%s': %s", fields[1], err)
			c.Header("WWW-Authenticate", realm)
			c.Render(401, render.JSON{Data: map[string]string{"error": "Please Authenticate"}})
			c.Abort()
			return
		}
		creds := strings.SplitN(string(auth), ":", 2)
		if user != creds[0] || password != creds[1] {
			c.Header("WWW-Authenticate", realm)
			c.Render(401, render.JSON{Data: map[string]string{"error": "Please Authenticate"}})
			c.Abort()
			return
		}
	}
}
