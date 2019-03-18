// +build generate

// Takes the scopes.json file and generates scopes.ts and scopes.go
package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var goTemplate = template.Must(template.New("go").Parse(`// Code generated by scopes_generate.go; DO NOT EDIT.

package {{.Language}}

// map of PoS to (map of Form to Lemma)
var Dictionary = map[string]map[string]string{
{{- range  $pos, $dict := .Entries}}
	"{{$pos}}": {
		{{- range  $f, $l := $dict}}
		"{{$f}}": "{{$l}}", {{end}}
	}, {{end}}
}
`))

// Dicts is a dictionary of posCodes-Dict
type Dicts map[string]Dict

// Dict is a dictionary of form-lemma relations
type Dict map[string]string

// remoceAccents removes accents from the string
// See https://blog.golang.org/normalization
func removeAccents(original string) (modified string, err error) {
	isMn := func(r rune) bool {
		return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
	}
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	modified, _, err = transform.String(t, original)
	return modified, err
}

func processEntry(langDicts Dicts, entry string) error {
	sEntry := strings.Split(entry, " ") // form lemma pos
	if len(sEntry) != 3 {
		return fmt.Errorf("Invalid entry %s", entry)
	}
	var dictKey string
	switch string(sEntry[2][0]) { // First character of pos
	case "D": // determiner
		dictKey = "DET"
	case "A": // adjective
		dictKey = "ADJ"
	case "N": // noun
		dictKey = "NOUN"
	case "V": // verb
		dictKey = "VERB"
	case "R": // adverb
		dictKey = "ADV"
	case "S": // adposition
		dictKey = "ADP"
	case "C": // conjuntion
		dictKey = "CONJ"
	case "P": // pronoun
		dictKey = "PRON"
	case "I": // interjection
		dictKey = "INTJ"
	default:
		return nil // Skip it
	}
	dict, ok := langDicts[dictKey]
	if !ok {
		dict = make(Dict)
	}

	if _, ok := dict[sEntry[0]]; !ok { // dont override, use first match
		dict[sEntry[0]] = sEntry[1]
		modified, err := removeAccents(sEntry[0])
		if err != nil {
			return err
		}
		if modified != sEntry[0] { // instance modified, it had accents. Try to add the corrected one
			if _, ok := dict[modified]; !ok { // dont override, use first match
				dict[modified] = sEntry[1]
			}
		}
	}
	langDicts[dictKey] = dict
	return nil
}

type LanguageDictionary struct {
	Language string
	Entries  Dicts
}

func loadDict(langDicts Dicts, dictFileName string) error {
	content, err := ioutil.ReadFile(dictFileName)
	if err != nil {
		return err
	}
	entries := strings.Split(string(content), "\n")
	for _, entry := range entries {
		if entry != "" {
			err := processEntry(langDicts, entry)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func generateLangDict(Language string, files []string) error {
	dicts := make(Dicts)
	for _, d := range files {
		err := loadDict(dicts, d)
		if err != nil {
			return err
		}
	}
	var langDict = LanguageDictionary{
		Language,
		dicts,
	}
	outFile := fmt.Sprintf("%v/dictionary.go", Language)
	langDictf, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	if err = goTemplate.Execute(langDictf, langDict); err != nil {
		return fmt.Errorf("render %v: %v", outFile, err)
	}
	return nil
}

func main() {
	fmt.Println("Starting dictionaries generation...")
	fmt.Println("[Corrector] Loading dictionaries...")
	esFiles := []string{
		"./data/es/MM.adj",
		"./data/es/MM.adv",
		"./data/es/MM.int",
		"./data/es/MM.nom",
		"./data/es/MM.tanc",
		"./data/es/MM.vaux",
		"./data/es/MM.verb",
	}
	err := generateLangDict("es", esFiles)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("[Corrector] Dictionaries loaded.")
}
