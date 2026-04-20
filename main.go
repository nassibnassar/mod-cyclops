package main

import "os"
import "strconv"
import "fmt"
import "regexp"
import "github.com/MikeTaylor/catlogger"
import "github.com/indexdata/ccms"
import "github.com/indexdata/mod-cyclops/cyclops"

// PRIVATE to this file
type config struct {
	logger           *catlogger.Logger
	queryTimeout     int
	serverHost       string
	serverPort       int
	ccmsClient       *ccms.Client
}

func makeCCMSClient(logger *catlogger.Logger) *ccms.Client {
	host := os.Getenv("CCMS_HOST")
	port := os.Getenv("CCMS_PORT")
	if port == "" {
		port = "8504"
	}
	user := os.Getenv("CCMS_USER")
	password := os.Getenv("CCMS_PASSWORD")

	logger.Log("client", fmt.Sprintf("host='%s' port='%s' user='%s' password='%s'", host, port, user, password))
	client := ccms.Client{
		Host:          host,
		Port:          port,
		User:          user,
		Password:      password,
		NoTLS:         true, // insecure, but server does not yet support TLS
		TLSSkipVerify: true, // likewise
	}

	return &client
}

func buildConfigFromEnv() *config {
	var cfg config

	loggingPrefix := os.Getenv("LOGGING_PREFIX")

	var loggingTimestamp bool = false
	tsString := os.Getenv("LOGGING_TIMESTAMP")
	if tsString != "" {
		loggingTimestamp = true
	}

	// catlogger.MakeLogger handes the category environment variables on its own
	cfg.logger = catlogger.MakeLogger("", loggingPrefix, loggingTimestamp)

	// We do not need this transformation yet, but will need something like it when we have authentication
	cfg.logger.AddTransformation(regexp.MustCompile(`\\"pass\\":\\"[^"]*\\"`), `\"pass\":\"********\"`)

	timeoutString := os.Getenv("MOD_CYCLOPS_QUERY_TIMEOUT")
	if timeoutString != "" {
		cfg.queryTimeout, _ = strconv.Atoi(timeoutString)
	} else {
		cfg.queryTimeout = 60
	}

	cfg.serverHost = os.Getenv("SERVER_HOST")
	if cfg.serverHost == "" {
		cfg.serverHost = "0.0.0.0"
	}

	serverPortString := os.Getenv("SERVER_PORT")
	if serverPortString != "" {
		cfg.serverPort, _ = strconv.Atoi(serverPortString)
	} else {
		cfg.serverPort = 12370
	}

	cfg.ccmsClient = makeCCMSClient(cfg.logger)

	return &cfg
}

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0])
		os.Exit(1)
	}

	cfg := buildConfigFromEnv()
	if cfg.ccmsClient.Host == "" {
		fmt.Fprintf(os.Stderr, "%s: bad configuration (no CCMS server)\n", os.Args[0])
		os.Exit(2)
	}

	cfg.logger.Log("config", fmt.Sprintf("%+v", cfg));

	server := cyclops.MakeModCyclopsServer(cfg.logger, cfg.ccmsClient, ".", cfg.queryTimeout)
	err := server.Launch(cfg.serverHost, cfg.serverPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot launch server: %s\n", os.Args[0], err)
		os.Exit(3)
	}
}
