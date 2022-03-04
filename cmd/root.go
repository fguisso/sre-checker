package cmd

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	notifyEmail       string
	checkInterval     time.Duration
	timeout           time.Duration
	healthThreshold   int32
	unhealthThreshold int32

	tcpHost string
	tcpPort string

	httpHost string
	httpPort string

	auth string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sre-checker",
	Short: "Check status from Tonto services",
	Run: func(cmd *cobra.Command, args []string) {
		tracking(cmd)
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

	rootCmd.PersistentFlags().StringVar(&tcpHost, "tcp-host", "", "TCP server host to be track")
	rootCmd.PersistentFlags().StringVar(&tcpPort, "tcp-port", "80", "TCP server port to be track")
	rootCmd.PersistentFlags().StringVar(&httpHost, "http-host", "", "HTTP server host to be track")
	rootCmd.PersistentFlags().StringVar(&httpPort, "http-port", "80", "HTTP server port to be track")
	rootCmd.PersistentFlags().StringVar(&auth, "auth", "", "Authentication token from server")
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

// testTontoHTTP make a GET request to Tonto service and check de response to
// determine the health of service.
func testTontoHTTP(host, port, authToken string) bool {
	resp, err := http.Get("https://" + host + ":" + port + "/?auth=" + authToken + "&buf=testing")
	if err != nil {
		fmt.Println("HTTP GET request error:", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("Unexpect response status:", resp.Status)
		return false
	}

	scanner := bufio.NewScanner(resp.Body)
	for i := 0; scanner.Scan() && i < 5; i++ {
		if strings.Contains(scanner.Text(), "CLOUDWALK") {
			return true
		}
		fmt.Println("Unexpected response message:", scanner.Text())
		return false
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Scan message error:", err)
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
		fmt.Printf("Wrong address: %v \n", err)
		return false
	}

	// Create the connection.
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Printf("Service offline: %v \n", err)
		return false
	}
	defer conn.Close()

	// Write the authentication.
	_, err = conn.Write([]byte("auth " + authToken))
	if err != nil {
		fmt.Printf("Write authentication failure: %v \n", err)
		return false
	}

	authReply := make([]byte, 1024)
	testReply := make([]byte, 1024)

	_, err = conn.Read(authReply)
	if err != nil {
		fmt.Printf("Reads message error: %v \n", err)
		return false
	}

	// If auth works, sends testing message.
	if strings.Contains(string(authReply), "ok") {
		_, err = conn.Write([]byte("Testing"))
		if err != nil {
			fmt.Printf("Write authentication failure: %v \n", err)
			return false
		}

		_, err = conn.Read(testReply)
		if err != nil {
			fmt.Printf("Reads message error: %v \n", err)
			return false
		}

		if strings.Contains(string(testReply), "CLOUDWALK") {
			return true
		}
	}

	return false
}

func tracking(cmd *cobra.Command) {
	checkInterval, _ := cmd.Flags().GetDuration("check-interval")
	tcpHost, _ := cmd.Flags().GetString("tcp-host")
	tcpPort, _ := cmd.Flags().GetString("tcp-port")
	httpHost, _ := cmd.Flags().GetString("http-host")
	httpPort, _ := cmd.Flags().GetString("http-port")
	auth := viper.GetString("TONTO_AUTH")
	healthThresold, _ := cmd.Flags().GetInt32("health-thresold")
	unhealthThresold, _ := cmd.Flags().GetInt32("unhealth-thresold")

	// tcpStatus := make(map[string]bool)
	tcpHealthCount := 0
	tcpUnhealthCount := 0
	httpHealthCount := 0
	httpUnhealthCount := 0

	fmt.Println("Starting tracking...")

	go func() {
		for {
			testResult := testTontoTCP(tcpHost, tcpPort, auth)
			// tcpStatus[time.Now().String()] = testResult
			// TODO: tcpStatus out of memory, persist data or something

			if testResult {
				tcpHealthCount++
				tcpUnhealthCount = 0
			} else {
				tcpUnhealthCount++
				tcpHealthCount = 0
			}

			if tcpHealthCount >= int(healthThresold) {
				tcpHealthCount = int(healthThresold)
				fmt.Println("Uptime")
			}
			if tcpUnhealthCount >= int(unhealthThresold) {
				tcpUnhealthCount = int(unhealthThresold)
				fmt.Println("Downtime")
			}

			fmt.Printf("Running... health: %v , unhealth: %v \n", tcpHealthCount, tcpUnhealthCount)

			time.Sleep(checkInterval)
		}
	}()

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
			fmt.Println("Uptime")
		}
		if httpUnhealthCount >= int(unhealthThresold) {
			httpUnhealthCount = int(unhealthThresold)
			fmt.Println("Downtime")
		}

		fmt.Printf("Running... health: %v , unhealth: %v \n", httpHealthCount, httpUnhealthCount)

		time.Sleep(checkInterval)
	}
}
