package main

import (
	"fmt"
	"strings"
)

// renderDetailView renderiza o drill-down do item selecionado.
func renderDetailView(m model) string {
	switch m.activeTab {
	case TabServices:
		return renderServiceDetail(m)
	case TabNodes:
		return renderNodeDetail(m)
	case TabAlerts:
		return renderAlertDetail(m)
	case TabRecommendations:
		return renderRecDetail(m)
	case TabAgents:
		return renderAgentDetail(m)
	case TabTasks:
		return renderTaskDetail(m)
	}
	return ""
}

func renderServiceDetail(m model) string {
	for _, s := range mockServices {
		if s.name != m.selectedItem {
			continue
		}
		var sb strings.Builder
		sb.WriteString(sTitle.Render("Service: " + s.name))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf("Status: %s   Replicas: %s\n\n",
			sSuccess.Render(s.status), s.replicas))
		sb.WriteString("Resources:\n")
		sb.WriteString(fmt.Sprintf("  CPU:  %s%%   Limit: 2 cores\n", fmt.Sprintf("%.1f", s.cpu)))
		sb.WriteString(fmt.Sprintf("  MEM:  %s%%   Limit: 4Gi\n", fmt.Sprintf("%.1f", s.mem)))
		sb.WriteString("\nCPU Trend (last 60s):\n")
		sb.WriteString("  " + sparkline(s.spark, 50))
		sb.WriteString("\n\n")
		sb.WriteString(sMuted.Render("Actions: [a] apply  [d] delete  [e] edit  [l] logs  [s] shell  [y] yaml"))
		return sb.String()
	}
	return ""
}

func renderNodeDetail(m model) string {
	for _, n := range mockNodes {
		if n.id != m.selectedItem {
			continue
		}
		var sb strings.Builder
		sb.WriteString(sTitle.Render("Node: " + n.id))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf("Hostname: %s   Role: %s   Status: %s\n\n",
			n.hostname, n.role, sSuccess.Render(n.status)))
		sb.WriteString("Resources:\n")
		sb.WriteString(fmt.Sprintf("  CPU:  %.1f%%   Memory: %.1f%%   Disk: %.1f%%\n\n", n.cpu, n.mem, n.disk))
		sb.WriteString(sMuted.Render("Actions: [d] drain  [l] agent logs  [r] restart agent"))
		return sb.String()
	}
	return ""
}

func renderAlertDetail(m model) string {
	for _, a := range mockAlerts {
		if a.service != m.selectedItem {
			continue
		}
		var sb strings.Builder
		sb.WriteString(sTitle.Render("Alert: " + a.level + " — " + a.service))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf("Level: %s   Service: %s   Time: %s\n\n",
			a.level, a.service, a.time))
		sb.WriteString("Message:\n  " + a.message + "\n")
		return sb.String()
	}
	return ""
}

func renderRecDetail(m model) string {
	for _, r := range mockRecs {
		if r.service != m.selectedItem {
			continue
		}
		var sb strings.Builder
		sb.WriteString(sTitle.Render("Recommendation: " + r.service))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf("Risk: %s   Tier: %s\n\n", r.risk, r.tier))
		sb.WriteString(fmt.Sprintf("CPU: %s   MEM: %s\n\n", r.cpu, r.mem))
		sb.WriteString("Reason:\n  " + r.reason + "\n\n")
		sb.WriteString(sMuted.Render("Actions: [a] apply  [d] dismiss"))
		return sb.String()
	}
	return ""
}

func renderAgentDetail(m model) string {
	for _, a := range mockAgents {
		if a.nodeID != m.selectedItem {
			continue
		}
		var sb strings.Builder
		sb.WriteString(sTitle.Render("Agent: " + a.nodeID))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf("Status: %s   Version: %s   Last seen: %s\n",
			sSuccess.Render(a.status), a.version, a.lastSeen))
		sb.WriteString(fmt.Sprintf("Services monitored: %d\n\n", a.services))
		sb.WriteString(sMuted.Render("Actions: [r] restart  [l] logs  [d] drain node"))
		return sb.String()
	}
	return ""
}

func renderTaskDetail(m model) string {
	for _, t := range mockTasks {
		if t.id != m.selectedItem {
			continue
		}
		var sb strings.Builder
		sb.WriteString(sTitle.Render("Task: " + t.id))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf("Service: %s   Node: %s\n", t.service, t.node))
		sb.WriteString(fmt.Sprintf("Status: %s   Desired: %s   Uptime: %s\n\n",
			sSuccess.Render(t.status), t.desired, t.uptime))
		sb.WriteString(sMuted.Render("Actions: [l] logs  [r] restart  [d] remove"))
		return sb.String()
	}
	return ""
}
