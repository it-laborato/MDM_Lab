// Package spec contains functionality to parse "MDMlab specs" yaml files
// (which are concatenated yaml files) that can be applied to a MDMlab server.
package spec

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-multierror"
)

var yamlSeparator = regexp.MustCompile(`(?m:^---[\t ]*)`)

// Group holds a set of "specs" that can be applied to a MDMlab server.
type Group struct {
	Queries  []*mdmlab.QuerySpec
	Teams    []json.RawMessage
	Packs    []*mdmlab.PackSpec
	Labels   []*mdmlab.LabelSpec
	Policies []*mdmlab.PolicySpec
	Software []*mdmlab.SoftwarePackageSpec
	// This needs to be interface{} to allow for the patch logic. Otherwise we send a request that looks to the
	// server like the user explicitly set the zero values.
	AppConfig              interface{}
	EnrollSecret           *mdmlab.EnrollSecretSpec
	UsersRoles             *mdmlab.UsersRoleSpec
	TeamsDryRunAssumptions *mdmlab.TeamSpecsDryRunAssumptions
}

// Metadata holds the metadata for a single YAML section/item.
type Metadata struct {
	Kind    string          `json:"kind"`
	Version string          `json:"apiVersion"`
	Spec    json.RawMessage `json:"spec"`
}

// GroupFromBytes parses a Group from concatenated YAML specs.
func GroupFromBytes(b []byte) (*Group, error) {
	specs := &Group{}
	for _, specItem := range SplitYaml(string(b)) {
		var s Metadata
		if err := yaml.Unmarshal([]byte(specItem), &s); err != nil {
			return nil, fmt.Errorf("failed to unmarshal spec item %w: \n%s", err, specItem)
		}

		kind := strings.ToLower(s.Kind)

		if s.Spec == nil {
			if kind == "" {
				return nil, errors.New(`Missing required fields ("spec", "kind") on provided configuration.`)
			}
			return nil, fmt.Errorf(`Missing required fields ("spec") on provided %q configuration.`, s.Kind)
		}

		switch kind {
		case mdmlab.QueryKind:
			var querySpec *mdmlab.QuerySpec
			if err := yaml.Unmarshal(s.Spec, &querySpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Queries = append(specs.Queries, querySpec)

		case mdmlab.PackKind:
			var packSpec *mdmlab.PackSpec
			if err := yaml.Unmarshal(s.Spec, &packSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Packs = append(specs.Packs, packSpec)

		case mdmlab.LabelKind:
			var labelSpec *mdmlab.LabelSpec
			if err := yaml.Unmarshal(s.Spec, &labelSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Labels = append(specs.Labels, labelSpec)

		case mdmlab.PolicyKind:
			var policySpec *mdmlab.PolicySpec
			if err := yaml.Unmarshal(s.Spec, &policySpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Policies = append(specs.Policies, policySpec)

		case mdmlab.AppConfigKind:
			if specs.AppConfig != nil {
				return nil, errors.New("config defined twice in the same file")
			}

			var appConfigSpec interface{}
			if err := yaml.Unmarshal(s.Spec, &appConfigSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.AppConfig = appConfigSpec

		case mdmlab.EnrollSecretKind:
			if specs.AppConfig != nil {
				return nil, errors.New("enroll_secret defined twice in the same file")
			}

			var enrollSecretSpec *mdmlab.EnrollSecretSpec
			if err := yaml.Unmarshal(s.Spec, &enrollSecretSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.EnrollSecret = enrollSecretSpec

		case mdmlab.UserRolesKind:
			var userRoleSpec *mdmlab.UsersRoleSpec
			if err := yaml.Unmarshal(s.Spec, &userRoleSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.UsersRoles = userRoleSpec

		case mdmlab.TeamKind:
			// unmarshal to a raw map as we don't want to strip away unknown/invalid
			// fields at this point - that validation is done in the apply spec/teams
			// endpoint so that it is enforced for both the API and the CLI.
			rawTeam := make(map[string]json.RawMessage)
			if err := yaml.Unmarshal(s.Spec, &rawTeam); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Teams = append(specs.Teams, rawTeam["team"])

		default:
			return nil, fmt.Errorf("unknown kind %q", s.Kind)
		}
	}
	return specs, nil
}

// SplitYaml splits a text file into separate yaml documents divided by ---
func SplitYaml(in string) []string {
	var out []string
	for _, chunk := range yamlSeparator.Split(in, -1) {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		out = append(out, chunk)
	}
	return out
}

func generateRandomString(sizeBytes int) string {
	b := make([]byte, sizeBytes)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func ExpandEnv(s string) (string, error) {
	out, err := expandEnv(s, true)
	return out, err
}

// expandEnv expands environment variables for a gitops file.
// $ can be escaped with a backslash, e.g. \$VAR
// \$ can be escaped with another backslash, etc., e.g. \\\$VAR
// $FLEET_VAR_XXX will not be expanded. These variables are expanded on the server.
// If secretsMap is not nil, $FLEET_SECRET_XXX will be evaluated and put in the map
// If secretsMap is nil, $FLEET_SECRET_XXX will cause an error.
func expandEnv(s string, failOnSecret bool) (string, error) {
	// Generate a random escaping prefix that doesn't exist in s.
	var preventEscapingPrefix string
	for {
		preventEscapingPrefix = "PREVENT_ESCAPING_" + generateRandomString(8)
		if !strings.Contains(s, preventEscapingPrefix) {
			break
		}
	}

	s = escapeString(s, preventEscapingPrefix)
	var err *multierror.Error
	s = mdmlab.MaybeExpand(s, func(env string) (string, bool) {
		switch {
		case strings.HasPrefix(env, preventEscapingPrefix):
			return "$" + strings.TrimPrefix(env, preventEscapingPrefix), true
		case strings.HasPrefix(env, mdmlab.ServerVarPrefix):
			// Don't expand mdmlab vars -- they will be expanded on the server
			return "", false
		case strings.HasPrefix(env, mdmlab.ServerSecretPrefix):
			if failOnSecret {
				err = multierror.Append(err, fmt.Errorf("environment variables with %q prefix are only allowed in profiles and scripts: %q",
					mdmlab.ServerSecretPrefix, env))
			}
			return "", false
		}
		v, ok := os.LookupEnv(env)
		if !ok {
			err = multierror.Append(err, fmt.Errorf("environment variable %q not set", env))
			return "", false
		}
		return v, true
	})
	if err != nil {
		return "", err
	}
	return s, nil
}

func ExpandEnvBytes(b []byte) ([]byte, error) {
	s, err := ExpandEnv(string(b))
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

func ExpandEnvBytesIgnoreSecrets(b []byte) ([]byte, error) {
	s, err := expandEnv(string(b), false)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// LookupEnvSecrets only looks up FLEET_SECRET_XXX environment variables. Escaping is not supported.
// This is used for finding secrets in scripts only. The original string is not modified.
// A map of secret names to values is updated.
func LookupEnvSecrets(s string, secretsMap map[string]string) error {
	if secretsMap == nil {
		return errors.New("secretsMap cannot be nil")
	}
	var err *multierror.Error
	_ = mdmlab.MaybeExpand(s, func(env string) (string, bool) {
		if strings.HasPrefix(env, mdmlab.ServerSecretPrefix) {
			// lookup the secret and save it, but don't replace
			v, ok := os.LookupEnv(env)
			if !ok {
				err = multierror.Append(err, fmt.Errorf("environment variable %q not set", env))
				return "", false
			}
			secretsMap[env] = v
		}
		return "", false
	})
	if err != nil {
		return err
	}
	return nil
}

var escapePattern = regexp.MustCompile(`(\\+\$)`)

func escapeString(s string, preventEscapingPrefix string) string {
	return escapePattern.ReplaceAllStringFunc(s, func(match string) string {
		if len(match)%2 != 0 {
			return match
		}
		return strings.Repeat("\\", (len(match)/2)-1) + "$" + preventEscapingPrefix
	})
}
