package connect

import "strings"

func insertSSHProxyCommand(argv []string, proxy string) []string {
	if len(argv) == 0 || proxy == "" {
		return argv
	}
	out := make([]string, 0, len(argv)+2)
	out = append(out, argv[0])
	for i := 1; i < len(argv); i++ {
		if argv[i] == "-p" {
			out = append(out, "-o", "ProxyCommand="+proxy)
		}
		out = append(out, argv[i])
	}
	if len(argv) > 0 && argv[len(argv)-1] != "" {
		// If -p was missing, still add proxy before target.
		if !containsArg(out, "ProxyCommand=") {
			out = append(out, "-o", "ProxyCommand="+proxy)
		}
	}
	return out
}

func containsArg(args []string, prefix string) bool {
	for _, a := range args {
		if strings.HasPrefix(a, prefix) {
			return true
		}
	}
	return false
}
