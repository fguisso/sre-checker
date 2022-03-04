package cmd

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/feeds"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type status struct {
	// tcpStatus = make(map[string]bool)
	tcpStatus  string
	httpStatus string
}

var (
	cfgFile string

	notifyEmail       string
	checkInterval     time.Duration
	timeout           time.Duration
	healthThreshold   int32
	unhealthThreshold int32
	rssFeed           bool

	tcpHost string
	tcpPort string

	httpHost string
	httpPort string

	statusMtx sync.RWMutex
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sre-checker",
	Short: "Check status from Tonto services",
	Run: func(cmd *cobra.Command, args []string) {
		actualStatus := status{
			tcpStatus:  "WAITING FOR STATUS",
			httpStatus: "WAITING FOR STATUS",
		}

		tracking(cmd, &actualStatus)

		rssFeedServer, _ := cmd.Flags().GetBool("rss-feed")
		if rssFeedServer {
			initRSSFeed(&actualStatus)
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/sre-checker.yaml)")
	rootCmd.PersistentFlags().StringVar(&notifyEmail, "notify-email", "", "Email to get notifications")
	rootCmd.PersistentFlags().DurationVar(&checkInterval, "check-interval", 5*time.Second, "Check interval in seconds")
	rootCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "t", 30*time.Second, "Max timeout from service in seconds")
	rootCmd.PersistentFlags().Int32Var(&healthThreshold, "health-thresold", 5, "Consecutive success")
	rootCmd.PersistentFlags().Int32Var(&unhealthThreshold, "unhealth-thresold", 5, "Consecutive failures")
	rootCmd.PersistentFlags().BoolVar(&rssFeed, "rss-feed", false, "Running RSS Feed server.")

	rootCmd.PersistentFlags().StringVar(&tcpHost, "tcp-host", "", "TCP server host to be track")
	rootCmd.PersistentFlags().StringVar(&tcpPort, "tcp-port", "80", "TCP server port to be track")
	rootCmd.PersistentFlags().StringVar(&httpHost, "http-host", "", "HTTP server host to be track")
	rootCmd.PersistentFlags().StringVar(&httpPort, "http-port", "80", "HTTP server port to be track")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name "sre-checker" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yml")
		viper.SetConfigName("sre-checker")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// notifyEmail sends an email to informe actual service status.
func notifyByEmail(email, service, status string) {
	user := viper.GetString("SMTP_USER")
	password := viper.GetString("SMTP_PASSWORD")
	from := viper.GetString("SMTP_EMAIL")
	addr := viper.GetString("SMTP_ADDR")
	host := viper.GetString("SMTP_HOST")

	to := []string{
		email,
	}

	msg := []byte("From: sre-checker@sre.com\r\n" +
		"To: " + email + "\r\n" +
		"Subject: Tonto " + service + " is " + status + "\r\n\r\n" +
		"This is a status messager saying that Tonto " + service + " is " + status + "\r\n")

	auth := smtp.PlainAuth("", user, password, host)

	err := smtp.SendMail(addr, auth, from, to, msg)

	if err != nil {
		fmt.Println("Email error:", err)
	}

	fmt.Println("Email sent successfully", service)
}

// testTontoHTTP make a GET request to Tonto service and check de response to
// determine the health of service.
func testTontoHTTP(host, port, authToken string) bool {
	resp, err := http.Get("https://" + host + ":" + port + "/?auth=" + authToken + "&buf=testing")
	if err != nil {
		fmt.Println("HTTP: HTTP GET request error:", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("HTTP: Unexpect response status:", resp.Status)
		return false
	}

	scanner := bufio.NewScanner(resp.Body)
	for i := 0; scanner.Scan() && i < 5; i++ {
		if strings.Contains(scanner.Text(), "CLOUDWALK") {
			return true
		}
		fmt.Println("HTTP: Unexpected response message:", scanner.Text())
		return false
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("HTTP: Scan message error:", err)
		return false
	}

	return false
}

// testTontoTCP create a connection with Tonto TCP services and test
// the authentication and sends a message to test the echo, returns TRUE if
// all testing rules works.
func testTontoTCP(host, port, authToken string) bool {
	// Parse the address.
	tcpAddr, err := net.ResolveTCPAddr("tcp", host+":"+port)
	if err != nil {
		fmt.Println("TCP: Wrong address:", err)
		return false
	}

	// Create the connection.
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Println("TCP: Service offline:", err)
		return false
	}
	defer conn.Close()

	// Write the authentication.
	_, err = conn.Write([]byte("auth " + authToken))
	if err != nil {
		fmt.Println("TCP: Write authentication failure:", err)
		return false
	}

	authReply := make([]byte, 1024)
	testReply := make([]byte, 1024)

	_, err = conn.Read(authReply)
	if err != nil {
		fmt.Println("TCP: Reads message error:", err)
		return false
	}

	// If auth works, sends testing message.
	if strings.Contains(string(authReply), "ok") {
		_, err = conn.Write([]byte("Testing"))
		if err != nil {
			fmt.Println("TCP: Write authentication failure:", err)
			return false
		}

		_, err = conn.Read(testReply)
		if err != nil {
			fmt.Println("TCP: Reads message error:", err)
			return false
		}

		if strings.Contains(string(testReply), "CLOUDWALK") {
			return true
		}
	}

	return false
}

func tracking(cmd *cobra.Command, status *status) {
	tcpHealthCount := 0
	tcpUnhealthCount := 0
	httpHealthCount := 0
	httpUnhealthCount := 0

	email, _ := cmd.Flags().GetString("notify-email")
	checkInterval, _ := cmd.Flags().GetDuration("check-interval")
	tcpHost, _ := cmd.Flags().GetString("tcp-host")
	tcpPort, _ := cmd.Flags().GetString("tcp-port")
	httpHost, _ := cmd.Flags().GetString("http-host")
	httpPort, _ := cmd.Flags().GetString("http-port")
	auth := viper.GetString("TONTO_AUTH")
	healthThresold, _ := cmd.Flags().GetInt32("health-thresold")
	unhealthThresold, _ := cmd.Flags().GetInt32("unhealth-thresold")

	fmt.Println("Starting tracking...")

	go func() {
		for {
			testResult := testTontoTCP(tcpHost, tcpPort, auth)

			if testResult {
				tcpHealthCount++
				tcpUnhealthCount = 0
			} else {
				tcpUnhealthCount++
				tcpHealthCount = 0
			}

			if tcpHealthCount >= int(healthThresold) {
				tcpHealthCount = int(healthThresold)
				statusMtx.Lock()
				if status.tcpStatus != "UP" {
					status.tcpStatus = "UP"
					notifyByEmail(email, "TCP", status.tcpStatus)
				}
				statusMtx.Unlock()
			}
			if tcpUnhealthCount >= int(unhealthThresold) {
				tcpUnhealthCount = int(unhealthThresold)
				statusMtx.Lock()
				if status.tcpStatus != "DOWN" {
					status.tcpStatus = "DOWN"
					notifyByEmail(email, "TCP", status.tcpStatus)
				}
				statusMtx.Unlock()
			}

			fmt.Printf("TCP status count: health(%v) unhealth(%v) \n", tcpHealthCount, tcpUnhealthCount)
			time.Sleep(checkInterval)
		}
	}()

	go func() {
		for {
			testResult := testTontoHTTP(httpHost, httpPort, auth)

			if testResult {
				httpHealthCount++
				httpUnhealthCount = 0
			} else {
				httpUnhealthCount++
				httpHealthCount = 0
			}

			if httpHealthCount >= int(healthThresold) {
				httpHealthCount = int(healthThresold)
				statusMtx.Lock()
				if status.httpStatus != "UP" {
					status.httpStatus = "UP"
					notifyByEmail(email, "HTTP", status.httpStatus)
				}
				statusMtx.Unlock()
			}
			if httpUnhealthCount >= int(unhealthThresold) {
				httpUnhealthCount = int(unhealthThresold)
				statusMtx.Lock()
				if status.httpStatus != "DOWN" {
					status.httpStatus = "DOWN"
					notifyByEmail(email, "HTTP", status.httpStatus)
				}
				statusMtx.Unlock()
			}
			fmt.Printf("HTTP status count: health(%v) unhealth(%v) \n", httpHealthCount, httpUnhealthCount)
			time.Sleep(checkInterval)
		}
	}()

}

func initRSSFeed(status *status) {
	fmt.Println("Starting RSS Fedd server...")
	router := mux.NewRouter()
	router.HandleFunc("/rss", func(w http.ResponseWriter, r *http.Request) {
		// Create RSS Feed Header
		feed := &feeds.Feed{
			Title:       "Tonto Services Monitor",
			Link:        &feeds.Link{Href: "/rss"},
			Description: "This is RSS Feeds with status from Tonto services.",
		}

		// Append TCP status
		statusMtx.RLock()
		feed.Add(&feeds.Item{
			Title: "Tonto TCP Service is " + status.tcpStatus,
			Link:  &feeds.Link{Href: "tonto.cloudwalk.io:3000"},
		})
		statusMtx.RUnlock()

		// Append HTTP status
		statusMtx.RLock()
		feed.Add(&feeds.Item{
			Title: "Tonto HTTP Service is " + status.httpStatus,
			Link:  &feeds.Link{Href: "https://tonto-http.cloudwalk.io"},
		})
		statusMtx.RUnlock()

		w.Header().Set("Content-Type", "application/rss+xml;charset=UTF-8")
		w.WriteHeader(http.StatusOK)

		rssFeed := (&feeds.Rss{Feed: feed}).RssFeed()
		err := feeds.WriteXML(rssFeed, w)
		if err != nil {
			fmt.Println("Write XML error:", err)
		}
	}).Methods("GET")

	http.Handle("/", router)

	//start and listen to requests
	http.ListenAndServe("0.0.0.0:8080", router)

}
