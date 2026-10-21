package views

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
)

// Networks renders Docker network list, IPAM configurations, and connected endpoints (NET-01).
func Networks(s style.Sheet, items []engine.Network, cursor int, filter string, width, height int, now time.Time) string {
	if len(items) == 0 {
		return NetworkList(s, items, 0, filter, width, height, now)
	}
	cursor = max(0, min(cursor, len(items)-1))
	if width < tokens.DetailWidth {
		return NetworkList(s, items, cursor, filter, width, height, now)
	}
	left := width * 3 / 7
	right := width - left - tokens.Gap
	result := lipgloss.JoinHorizontal(lipgloss.Top,
		NetworkList(s, items, cursor, filter, left, height, now),
		strings.Repeat(" ", tokens.Gap),
		NetworkDetail(s, items[cursor], right, height, now),
	)
	return style.Fit(result, width, height)
}

// NetworkList renders the compact, sorted table of Docker networks.
func NetworkList(s style.Sheet, items []engine.Network, cursor int, filter string, width, height int, _ time.Time) string {
	lines := []string{s.Muted("filter " + emptyDash(filter))}
	capacity := max(1, height-tokens.PanelFrame-tokens.InsetY*2-1)
	start := max(0, cursor-capacity+1)
	end := min(len(items), start+capacity)
	g := tokens.Icons(s.ASCII)

	for n := start; n < end; n++ {
		net := items[n]
		marker := " "
		if n == cursor {
			marker = g.Cursor
		}
		driver := net.Driver
		if driver == "" {
			driver = "-"
		}
		epCount := fmt.Sprintf("%d endpoints", len(net.Containers))
		nameWidth := max(10, width-32)
		name := padCell(style.Clean(net.Name), nameWidth)
		line := fmt.Sprintf("%s %s %-8s %s", marker, name, style.Clean(driver), epCount)
		if n == cursor {
			line = s.Text(line, s.Palette.Accent)
		}
		lines = append(lines, line)
	}

	if len(items) == 0 {
		return s.Panel("NETWORKS  0", components.Empty(s, "No Docker networks found.", "Create one with c, or / to change the filter.", max(1, width-tokens.PanelFrame-tokens.InsetX*2)), width, height, true)
	}
	return s.Panel(fmt.Sprintf("NETWORKS  %d", len(items)), strings.Join(lines, "\n"), width, height, true)
}

// NetworkDetail renders network configuration, IPAM subnets, and attached containers.
func NetworkDetail(s style.Sheet, net engine.Network, width, height int, now time.Time) string {
	var lines []string
	header := style.Clean(net.Name)
	if net.Driver != "" {
		header += " · " + style.Clean(net.Driver)
	}
	lines = append(lines, header, engine.ShortID(net.ID), "")

	if !net.Created.IsZero() {
		lines = append(lines, "Created "+components.Age(net.Created, now), "")
	}

	var flags []string
	if net.Internal {
		flags = append(flags, "internal")
	}
	if net.Attachable {
		flags = append(flags, "attachable")
	}
	if net.Ingress {
		flags = append(flags, "ingress")
	}
	if net.IPv6 {
		flags = append(flags, "ipv6")
	}
	if len(flags) > 0 {
		lines = append(lines, s.Heading("FLAGS"), strings.Join(flags, " · "), "")
	}

	lines = append(lines, s.Heading("IPAM CONFIGURATION"))
	if len(net.Subnets) == 0 && len(net.Gateways) == 0 {
		lines = append(lines, "Default daemon address pool.")
	} else {
		for i, subnet := range net.Subnets {
			gw := "-"
			if i < len(net.Gateways) {
				gw = net.Gateways[i]
			}
			lines = append(lines, fmt.Sprintf("Subnet: %s   Gateway: %s", style.Clean(subnet), style.Clean(gw)))
		}
	}
	lines = append(lines, "")

	lines = append(lines, s.Heading(fmt.Sprintf("CONNECTED CONTAINERS (%d)", len(net.Containers))))
	if len(net.Containers) == 0 {
		lines = append(lines, "No containers attached. Safe to remove.")
	} else {
		for _, ep := range net.Containers {
			ip := ep.IPv4
			if ip == "" {
				ip = "no IPv4"
			}
			lines = append(lines, fmt.Sprintf("%-16s %-18s %s",
				style.Clean(ep.Name),
				style.Clean(ip),
				s.Muted(engine.ShortID(ep.ContainerID)),
			))
		}
	}

	lines = append(lines, "", s.Muted("c create · d remove · / filter"))
	body := strings.Join(lines, "\n")
	return s.Panel("NETWORK DETAIL", body, width, height, false)
}
