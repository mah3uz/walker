package polkit

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	agentInterface = "org.freedesktop.PolicyKit1.AuthenticationAgent"
	agentPath      = "/org/freedesktop/PolicyKit1/AuthenticationAgent"
	agentBusName   = "org.example.PolicyKit1.AuthenticationAgent"
)

type Agent struct {
	conn *dbus.Conn
}

// Subject represents a PolicyKit subject
type Subject struct {
	Kind    string
	Details map[string]dbus.Variant
}

func getPassword() (string, error) {
	cmd := exec.Command("walker", "--password")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get password: %v", err)
	}

	return string(out), nil
}

func verifyPassword(username, password string) bool {
	cmd := exec.Command("su", username, "-c", "true")
	cmd.Stdin = strings.NewReader(password + "\n")
	err := cmd.Run()
	if err != nil {
		log.Printf("Password verification failed: %v", err)
		return false
	}
	return true
}

// BeginAuthentication handles the authentication request
func (a *Agent) BeginAuthentication(actionId string, message string, iconName string, details map[string]string, cookie string, identities []interface{}) *dbus.Error {
	log.Printf("Authentication requested for action: %s\n", actionId)
	log.Printf("Message: %s\n", message)
	log.Printf("Cookie: %s\n", cookie)

	currentUser := os.Getenv("SUDO_USER")
	if currentUser == "" {
		currentUser = os.Getenv("USER")
	}
	if currentUser == "" {
		log.Printf("Could not determine user")
		return dbus.MakeFailedError(fmt.Errorf("could not determine user"))
	}

	log.Printf("Authenticating as user: %s", currentUser)

	userInfo, err := user.Lookup(currentUser)
	if err != nil {
		log.Printf("Failed to lookup user: %v", err)
		return dbus.MakeFailedError(err)
	}

	uid, err := strconv.ParseUint(userInfo.Uid, 10, 32)
	if err != nil {
		log.Printf("Failed to parse UID: %v", err)
		return dbus.MakeFailedError(err)
	}

	password, err := getPassword()
	if err != nil {
		log.Printf("Failed to get password: %v", err)
		return dbus.MakeFailedError(err)
	}

	if !verifyPassword(currentUser, password) {
		log.Printf("Invalid password for user %s", currentUser)
		return dbus.MakeFailedError(fmt.Errorf("invalid password"))
	}

	log.Printf("Password verified for user %s (uid: %d)", currentUser, uid)

	// Create the identity structure in the format PolicyKit expects: (sa{sv})
	identity := struct {
		Kind    string
		Details map[string]dbus.Variant
	}{
		Kind: "unix-user",
		Details: map[string]dbus.Variant{
			"uid": dbus.MakeVariant(uint32(uid)),
		},
	}

	// Send authentication response
	obj := a.conn.Object("org.freedesktop.PolicyKit1", "/org/freedesktop/PolicyKit1/Authority")
	call := obj.Call("org.freedesktop.PolicyKit1.Authority.AuthenticationAgentResponse2", 0,
		uint32(uid), // u
		cookie,      // s
		identity,    // (sa{sv})
	)

	if call.Err != nil {
		log.Printf("Failed to send authentication response: %v", call.Err)
		return dbus.MakeFailedError(call.Err)
	}

	log.Println("Authentication response sent successfully")
	return nil
}

func (a *Agent) CancelAuthentication(cookie string) *dbus.Error {
	log.Printf("Authentication cancelled for cookie: %s\n", cookie)
	return nil
}

func getCurrentSession() (string, error) {
	if session := os.Getenv("XDG_SESSION_ID"); session != "" {
		return session, nil
	}

	cmd := exec.Command("loginctl", "show-session", "self", "--property=Id")
	output, err := cmd.Output()
	if err == nil {
		session := strings.TrimPrefix(strings.TrimSpace(string(output)), "Id=")
		return session, nil
	}

	cmd = exec.Command("loginctl", "list-sessions", "--no-legend")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get session: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 {
		fields := strings.Fields(lines[0])
		if len(fields) > 0 {
			return fields[0], nil
		}
	}

	return "", fmt.Errorf("no session found")
}

func main() {
	logFile, err := os.OpenFile("/tmp/polkit-agent.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	conn, err := dbus.SystemBus()
	if err != nil {
		log.Fatalf("Failed to connect to system bus: %v", err)
	}

	reply, err := conn.RequestName(agentBusName,
		dbus.NameFlagDoNotQueue)
	if err != nil {
		log.Fatalf("Failed to request name: %v", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Fatal("Name already taken")
	}

	agent := &Agent{conn: conn}
	err = conn.Export(agent, dbus.ObjectPath(agentPath), agentInterface)
	if err != nil {
		log.Fatalf("Failed to export agent: %v", err)
	}

	sessionId, err := getCurrentSession()
	if err != nil {
		log.Fatalf("Failed to get current session: %v", err)
	}
	log.Printf("Using session ID: %s", sessionId)

	// Create the subject structure exactly as PolicyKit expects
	subject := Subject{
		Kind: "unix-session",
		Details: map[string]dbus.Variant{
			"session-id": dbus.MakeVariant(sessionId),
		},
	}

	obj := conn.Object("org.freedesktop.PolicyKit1", "/org/freedesktop/PolicyKit1/Authority")
	call := obj.Call("org.freedesktop.PolicyKit1.Authority.RegisterAuthenticationAgent", 0,
		subject,
		"en_US.UTF-8",
		agentPath,
	)

	if call.Err != nil {
		log.Fatalf("Failed to register authentication agent: %v", call.Err)
	}

	// Also register with options
	call = obj.Call("org.freedesktop.PolicyKit1.Authority.RegisterAuthenticationAgentWithOptions", 0,
		subject,
		"en_US.UTF-8",
		agentPath,
		map[string]dbus.Variant{},
	)

	if call.Err != nil {
		log.Printf("Warning: Failed to register with options: %v", call.Err)
	}

	log.Println("Successfully registered authentication agent")
	fmt.Println("PolicyKit agent started. Waiting for authentication requests...")
	fmt.Println("Logs are being written to /tmp/polkit-agent.log")

	select {}
}
