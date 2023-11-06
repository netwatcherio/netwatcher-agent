package workers

import (
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/api"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"os/exec"
	"runtime"
	// other imports
)

// Other code...

func startCheckWorker(id primitive.ObjectID, dataChan chan api.CheckData) {
	// Your existing worker initialization code...
	go func(i primitive.ObjectID, dC chan api.CheckData) {
		for {
			// Your existing loop code...

			// Determine the executable to run based on OS
			var cmd *exec.Cmd
			switch runtime.GOOS {
			case "windows":
				cmd = exec.Command("path_to_windows_executable", "arg1", "arg2")
			case "darwin":
				cmd = exec.Command("path_to_mac_executable", "arg1", "arg2")
			case "linux":
				cmd = exec.Command("path_to_linux_executable", "arg1", "arg2")
			default:
				fmt.Printf("Unsupported operating system: %s\n", runtime.GOOS)
				// Handle unsupported operating systems.
			}

			// Run the command if it's set
			if cmd != nil {
				output, err := cmd.CombinedOutput()
				if err != nil {
					fmt.Printf("Failed to execute command: %s\n", err)
					// Handle error...
				}
				fmt.Printf("Output from command: %s\n", output)
			}

			// Your existing loop code...
		}
	}(id, dataChan)
}

// Other code...
