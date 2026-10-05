package egg

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"app/api/v1alpha1"
	"app/internal/validation"
)

// FieldErrors maps a field path (e.g. "variables.2.envVariable") to a message.
type FieldErrors map[string]string

func (f FieldErrors) Error() string { return validation.Summary(f, humanize) }

// Fields returns the messages per field path (validation.Error).
func (f FieldErrors) Fields() map[string]string { return f }

var fieldNames = map[string]string{
	"displayName":  "Name",
	"author":       "Author",
	"startup":      "Startup command",
	"dockerImages": "Image",
	"variables":    "Variable",
	"configFiles":  "Config file",
	"replace":      "replacement",
	"startupDone":  "Startup text",
	"envVariable":  "environment variable",
	"defaultValue": "default value",
	"rules":        "rules",
	"name":         "name",
	"image":        "image",
	"file":         "file",
	"parser":       "parser",
	"match":        "key",
	"valueType":    "type",
}

// humanize turns "variables.0.envVariable" into "Variable 1, environment variable".
func humanize(path string) string {
	var parts []string
	for _, seg := range strings.Split(path, ".") {
		if n, err := strconv.Atoi(seg); err == nil && len(parts) > 0 {
			parts[len(parts)-1] += " " + strconv.Itoa(n+1)
			continue
		}
		if name, ok := fieldNames[seg]; ok {
			seg = name
		}
		parts = append(parts, seg)
	}
	return strings.Join(parts, ", ")
}

// Same rule as Pterodactyl (Pelican also allows dashes, which shells do not accept in names).
var envNameRe = regexp.MustCompile(`^\w{1,191}$`)

// reservedForEdit extends ReservedVariables with the names Pelican reserves, so edited eggs stay
// importable there.
var reservedForEdit = []string{"MODIFIED_STARTUP", "INTERNAL_IP", "HOSTNAME", "TERM", "LANG", "PWD", "TIMEZONE"}

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

var parsers = []string{"file", "yaml", "properties", "ini", "json", "xml"}

// Normalize trims values and drops empty list entries before validation and saving.
func Normalize(s *v1alpha1.EggSpec) {
	s.DisplayName = strings.TrimSpace(s.DisplayName)
	s.Author = strings.TrimSpace(s.Author)
	s.Description = strings.TrimSpace(s.Description)
	s.Startup = strings.TrimSpace(s.Startup)
	s.Features = compact(s.Features)
	s.FileDenylist = compact(s.FileDenylist)
	s.StartupDone = compact(s.StartupDone)
	s.Install.Script = strings.ReplaceAll(s.Install.Script, "\r\n", "\n")
	s.Install.Container = strings.TrimSpace(s.Install.Container)
	s.Install.Entrypoint = strings.TrimSpace(s.Install.Entrypoint)
	for i := range s.DockerImages {
		s.DockerImages[i].Name = strings.TrimSpace(s.DockerImages[i].Name)
		s.DockerImages[i].Image = strings.TrimSpace(s.DockerImages[i].Image)
		if s.DockerImages[i].Name == "" {
			s.DockerImages[i].Name = s.DockerImages[i].Image
		}
	}
	for i := range s.Variables {
		v := &s.Variables[i]
		v.Name, v.EnvVariable, v.Rules = strings.TrimSpace(
			v.Name,
		), strings.TrimSpace(
			v.EnvVariable,
		), strings.TrimSpace(
			v.Rules,
		)
		if v.FieldType == "" {
			v.FieldType = "text"
		}
	}
	for i := range s.ConfigFiles {
		s.ConfigFiles[i].File = strings.TrimSpace(s.ConfigFiles[i].File)
	}
}

// ValidateSpec checks an egg edited in the panel. The rules follow Pterodactyl and Pelican, so
// exports stay importable there. Errors are keyed by field path, e.g. "variables.2.envVariable".
func ValidateSpec(s *v1alpha1.EggSpec) error {
	errs := FieldErrors{}
	validateGeneral(s, errs)
	validateImages(s.DockerImages, errs)
	validateVariables(s.Variables, errs)
	validateConfigFiles(s.ConfigFiles, errs)
	validateStartupDone(s.StartupDone, errs)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateGeneral(s *v1alpha1.EggSpec, errs FieldErrors) {
	switch {
	case s.DisplayName == "":
		errs["displayName"] = "is required"
	case len(s.DisplayName) > 191:
		errs["displayName"] = "must be at most 191 characters"
	}
	// Pterodactyl and Pelican require an e-mail address as author when importing.
	switch {
	case s.Author == "":
		errs["author"] = "is required (an e-mail address)"
	case !emailRe.MatchString(s.Author):
		errs["author"] = "must be an e-mail address"
	}
	if s.Startup == "" {
		errs["startup"] = "is required"
	}
}

func validateImages(images []v1alpha1.DockerImage, errs FieldErrors) {
	if len(images) == 0 {
		errs["dockerImages"] = "add at least one image"
	}
	seenName, seenImage := map[string]bool{}, map[string]bool{}
	for i, img := range images {
		p := fmt.Sprintf("dockerImages.%d.", i)
		switch {
		case img.Image == "":
			errs[p+"image"] = "is required"
		case strings.ContainsAny(img.Image, " \t\n"):
			errs[p+"image"] = "must not contain spaces"
		case seenImage[img.Image]:
			errs[p+"image"] = "is listed twice"
		case seenName[img.Name]:
			errs[p+"name"] = "is used twice"
		}
		seenName[img.Name], seenImage[img.Image] = true, true
	}
}

func validateVariables(vars []v1alpha1.EggVariable, errs FieldErrors) {
	seen := map[string]bool{}
	for i, v := range vars {
		p := fmt.Sprintf("variables.%d.", i)
		if v.Name == "" {
			errs[p+"name"] = "is required"
		}
		if msg := envVariableProblem(v.EnvVariable, seen); msg != "" {
			errs[p+"envVariable"] = msg
		}
		seen[v.EnvVariable] = true
		if err := checkRules(v.Rules); err != nil {
			errs[p+"rules"] = err.Error()
		}
	}
}

// envVariableProblem returns why an environment variable name cannot be used ("" when it can).
func envVariableProblem(name string, seen map[string]bool) string {
	upper := strings.ToUpper(name)
	switch {
	case !envNameRe.MatchString(name):
		return "letters, digits and underscores only"
	case ReservedVariables[upper] || slices.Contains(reservedForEdit, upper):
		return "is set by the panel and cannot be used"
	case seen[name]:
		return "is used twice"
	}
	return ""
}

var valueTypes = []string{"string", "number", "boolean"}

func validateConfigFiles(files []v1alpha1.ConfigFile, errs FieldErrors) {
	seen := map[string]bool{}
	for i, f := range files {
		p := fmt.Sprintf("configFiles.%d.", i)
		switch {
		case f.File == "":
			errs[p+"file"] = "is required"
		case strings.HasPrefix(f.File, "/") || slices.Contains(strings.Split(f.File, "/"), ".."):
			errs[p+"file"] = "must be a path inside the server folder"
		case seen[f.File]:
			errs[p+"file"] = "is listed twice"
		}
		seen[f.File] = true
		if !slices.Contains(parsers, f.Parser) {
			errs[p+"parser"] = "must be one of " + strings.Join(parsers, ", ")
		}
		for j, r := range f.Replace {
			rp := fmt.Sprintf("%sreplace.%d.", p, j)
			if strings.TrimSpace(r.Match) == "" {
				errs[rp+"match"] = "is required"
			}
			if r.ValueType != "" && !slices.Contains(valueTypes, r.ValueType) {
				errs[rp+"valueType"] = "must be string, number or boolean"
			}
		}
	}
}

// validateStartupDone checks the "regex:" entries of the startup detection.
func validateStartupDone(done []string, errs FieldErrors) {
	for i, d := range done {
		if re, ok := strings.CutPrefix(d, "regex:"); ok {
			if _, err := regexp.Compile(re); err != nil {
				errs[fmt.Sprintf("startupDone.%d", i)] = "invalid regular expression: " + err.Error()
			}
		}
	}
}

// checkRules rejects rule strings the panel cannot evaluate correctly (broken regular expressions,
// missing arguments).
func checkRules(rules string) error {
	for _, r := range SplitRules(rules) {
		name, arg, hasArg := strings.Cut(r, ":")
		switch name {
		case "regex", "not_regex":
			if _, err := phpRegex(arg); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		case "max", "min", "gt", "size", "between", "digits_between", "in", "not_in", "ends_with":
			if !hasArg || arg == "" {
				return fmt.Errorf("%s needs an argument (%s:…)", name, name)
			}
		}
	}
	return nil
}

func compact(list []string) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
