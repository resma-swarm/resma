package main

import (
	"fmt"
	"strings"
)

func renderDetailView(m model) string {
	var body string
	switch m.activeTab {
	case TabServices:
		body = renderServiceDetail(m)
	case TabNodes:
		body = renderNodeDetail(m)
	case TabAlerts:
		body = renderAlertDetail(m)
	case TabRecommendations:
		body = renderRecDetail(m)
	case TabAgents:
		body = renderAgentDetail(m)
	case TabTasks:
		body = renderTaskDetail(m)
	}

	return sK9sInfoVal.Width(m.width).Render(body)
}

func renderServiceDetail(m model) string {
	for _, s := range mockServices {
		if s.name != m.selectedItem {
			continue
		}
		var sb strings.Builder
		sb.WriteString(sK9sClusterTitle.Render(" SERVICE: " + s.name + " "))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf(" Status:   %s\n", sK9sGreen.Render(s.status)))
		sb.WriteString(fmt.Sprintf(" Replicas: %s\n", s.replicas))
		sb.WriteString("\n Resources:\n")
		sb.WriteString(fmt.Sprintf("   CPU:  %s%% (Limit: 2 cores)\n", fmt.Sprintf("%.1f", s.cpu)))
		sb.WriteString(fmt.Sprintf("   MEM:  %s%% (Limit: 4Gi)\n", fmt.Sprintf("%.1f", s.mem)))
		sb.WriteString("\n CPU Trend:\n")
		sb.WriteString("   " + sparkline(s.spark, 50))
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
		sb.WriteString(sK9sClusterTitle.Render(" NODE: " + n.id + " "))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf(" Hostname: %s\n", n.hostname))
		sb.WriteString(fmt.Sprintf(" Role:     %s\n", n.role))
		sb.WriteString(fmt.Sprintf(" Status:   %s\n", sK9sGreen.Render(n.status)))
		sb.WriteString("\n Resources:\n")
		sb.WriteString(fmt.Sprintf("   CPU:  %.1f%%\n", n.cpu))
		sb.WriteString(fmt.Sprintf("   MEM:  %.1f%%\n", n.mem))
		sb.WriteString(fmt.Sprintf("   DISK: %.1f%%\n", n.disk))
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
		sb.WriteString(sK9sClusterTitle.Render(" ALERT: " + a.service + " "))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf(" Level:   %s\n", a.level))
		sb.WriteString(fmt.Sprintf(" Time:    %s\n", a.time))
		sb.WriteString("\n Message:\n")
		sb.WriteString("   " + a.message + "\n")
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
		sb.WriteString(sK9sClusterTitle.Render(" RECOMMENDATION: " + r.service + " "))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf(" Risk: %s\n", r.risk))
		sb.WriteString(fmt.Sprintf(" Tier: %s\n", r.tier))
		sb.WriteString(fmt.Sprintf(" CPU:  %s\n", r.cpu))
		sb.WriteString(fmt.Sprintf(" MEM:  %s\n", r.mem))
		sb.WriteString("\n Reason:\n")
		sb.WriteString("   " + r.reason + "\n")
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
		sb.WriteString(sK9sClusterTitle.Render(" AGENT: " + a.nodeID + " "))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf(" Status:    %s\n", sK9sGreen.Render(a.status)))
		sb.WriteString(fmt.Sprintf(" Version:   %s\n", a.version))
		sb.WriteString(fmt.Sprintf(" Last Seen: %s\n", a.lastSeen))
		sb.WriteString(fmt.Sprintf(" Services:  %d\n", a.services))
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
		sb.WriteString(sK9sClusterTitle.Render(" TASK: " + t.id + " "))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf(" Service: %s\n", t.service))
		sb.WriteString(fmt.Sprintf(" Node:    %s\n", t.node))
		sb.WriteString(fmt.Sprintf(" Status:  %s\n", sK9sGreen.Render(t.status)))
		sb.WriteString(fmt.Sprintf(" Desired: %s\n", t.desired))
		sb.WriteString(fmt.Sprintf(" Uptime:  %s\n", t.uptime))
		return sb.String()
	}
	return ""
}
