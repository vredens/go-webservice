package webservice

import (
	"os"
	"os/user"
	"strings"
)

const (
	httpAgentPrefix = "go-webservice/v0"
)

// userAgent generates a User-Agent string that identifies the current process.
// It first checks for the presence of the "USER_AGENT" environment variable to construct
// the User-Agent string in the format: "go-webservice/v0/{USER_AGENT}".
// It then checks for the presence of the "SYSTEM" and "COMPONENT" environment variables
// to construct the User-Agent string in the format: "go-webservice/v0/{SYSTEM}/{COMPONENT}".
// If these variables are not set, it falls back to using the current user's information,
// such as their name, username, or home directory, in the format: "go-webservice/v0/{USER_INFO}".
// If no user information is available or an error occurs, it returns "UNKNOWN".
func userAgent() string {
	var agent = strings.Builder{}

	agent.WriteString(httpAgentPrefix)

	if ua := os.Getenv("USER_AGENT"); ua != "" {
		agent.WriteRune('/')
		agent.WriteString(ua)
		return agent.String()
	}

	var system = os.Getenv("SYSTEM")
	var component = os.Getenv("COMPONENT")
	if system != "" {
		agent.WriteRune('/')
		agent.WriteString(system)
		if component != "" {
			agent.WriteRune('/')
			agent.WriteString(component)
		}

		return agent.String()
	}

	var usr, err = user.Current()
	if err != nil {
		return "UNKNOWN"
	}
	if usr.Name != "" {
		agent.WriteRune('/')
		agent.WriteString(usr.Name)
		return agent.String()
	}
	if usr.Username != "" {
		agent.WriteRune('/')
		agent.WriteString(usr.Username)
		return agent.String()
	}
	if usr.HomeDir != "" {
		agent.WriteRune('/')
		agent.WriteString(usr.HomeDir)
		return agent.String()
	}

	return "UNKNOWN"
}

func combineURL(base string, endpoint string) string {
	if strings.HasSuffix(base, "/") {
		if strings.HasPrefix(endpoint, "/") {
			return base + endpoint[1:]
		}
		return base + endpoint
	}
	if strings.HasPrefix(endpoint, "/") {
		return base + endpoint
	}
	return base + "/" + endpoint
}
