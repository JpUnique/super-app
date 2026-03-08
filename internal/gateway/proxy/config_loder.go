package proxy

import (
    "fmt"
    "os"
    "sort"

    "gopkg.in/yaml.v3"
)

// YAML schema
type fileRoutes struct {
    Services map[string]serviceConfig `yaml:"services"`
    Routes   []fileRoute              `yaml:"routes"`
}

type serviceConfig struct {
    Strategy string `yaml:"strategy"` // "round_robin" | "least_conn"
}

type fileRoute struct {
    Prefix      string `yaml:"prefix"`
    Service     string `yaml:"service"`
    StripPrefix bool   `yaml:"strip_prefix"`
}

// LoadRoutesFromYAML builds a StaticServiceRouter and returns a map[service]strategy.
func LoadRoutesFromYAML(path string) (*StaticServiceRouter, map[string]string, error) {
    b, err := os.ReadFile(path)
    if err != nil {
        return nil, nil, fmt.Errorf("read routes file: %w", err)
    }
    var fr fileRoutes
    if err := yaml.Unmarshal(b, &fr); err != nil {
        return nil, nil, fmt.Errorf("parse yaml: %w", err)
    }

    // Build StaticServiceRouter from routes
    routes := make([]Route, 0, len(fr.Routes))
    for _, r := range fr.Routes {
        if r.Prefix == "" || r.Service == "" {
            return nil, nil, fmt.Errorf("invalid route: prefix=%q service=%q", r.Prefix, r.Service)
        }
        routes = append(routes, Route{
            Prefix:      r.Prefix,
            ServiceName: r.Service,
            StripPrefix: r.StripPrefix,
        })
    }
    sr := NewStaticServiceRouter(routes)

    // Normalize strategies: default to round_robin
    strats := map[string]string{}
    for svc, sc := range fr.Services {
        switch sc.Strategy {
        case "", "round_robin", "least_conn":
            if sc.Strategy == "" {
                strats[svc] = "round_robin"
            } else {
                strats[svc] = sc.Strategy
            }
        default:
            return nil, nil, fmt.Errorf("unknown strategy %q for service %q", sc.Strategy, svc)
        }
    }
    // Ensure all referenced services have an entry
    for _, svc := range sr.Services() {
        if _, ok := strats[svc]; !ok {
            strats[svc] = "round_robin"
        }
    }
    return sr, strats, nil
}

// (helpers to list services/prefixes if your router.go doesn't have them)
func (sr *StaticServiceRouter) Services() []string {
    seen := map[string]struct{}{}
    out := make([]string, 0, len(sr.routes))
    for _, r := range sr.routes {
        if _, ok := seen[r.ServiceName]; !ok {
            seen[r.ServiceName] = struct{}{}
            out = append(out, r.ServiceName)
        }
    }
    sort.Strings(out)
    return out
}

func (sr *StaticServiceRouter) Prefixes() []string {
    seen := map[string]struct{}{}
    out := make([]string, 0, len(sr.routes))
    for _, r := range sr.routes {
        if _, ok := seen[r.Prefix]; !ok {
            seen[r.Prefix] = struct{}{}
            out = append(out, r.Prefix)
        }
    }
    sort.Strings(out)
    return out
}