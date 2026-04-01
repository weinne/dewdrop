package bootstrap

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type DependencyReport struct {
	Missing []string
	Details []string
}

func (r DependencyReport) Error() error {
	if len(r.Missing) == 0 && len(r.Details) == 0 {
		return nil
	}

	parts := make([]string, 0, len(r.Missing)+len(r.Details)+1)
	if len(r.Missing) > 0 {
		parts = append(parts, "dependencias ausentes: "+strings.Join(r.Missing, ", "))
	}
	parts = append(parts, r.Details...)

	return errors.New(strings.Join(parts, "; "))
}

func CheckRuntimeDependencies() error {
	report := DependencyReport{}

	if _, err := exec.LookPath("rclone"); err != nil {
		report.Missing = append(report.Missing, "rclone")
	}

	if _, err := exec.LookPath("systemctl"); err != nil {
		report.Missing = append(report.Missing, "systemd")
	}

	if _, err := exec.LookPath("fusermount3"); err != nil {
		if _, err2 := exec.LookPath("fusermount"); err2 != nil {
			report.Missing = append(report.Missing, "fuse3")
		}
	}

	if len(report.Missing) > 0 {
		return report.Error()
	}

	cmd := exec.Command("systemctl", "--user", "is-system-running")
	if out, err := cmd.CombinedOutput(); err != nil {
		state := strings.TrimSpace(string(out))
		if state == "" {
			state = err.Error()
		}

		// "degraded" is a valid and common state where user units still work.
		if state != "degraded" {
			probe := exec.Command("systemctl", "--user", "show-environment")
			if probeOut, probeErr := probe.CombinedOutput(); probeErr != nil {
				probeMsg := strings.TrimSpace(string(probeOut))
				if probeMsg == "" {
					probeMsg = probeErr.Error()
				}
				report.Details = append(report.Details, fmt.Sprintf("systemd --user indisponivel: %s", probeMsg))
			}
		}
	}

	return report.Error()
}
