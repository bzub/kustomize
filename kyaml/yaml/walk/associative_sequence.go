// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package walk

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-errors/errors"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func (l *Walker) walkAssociativeSequence() (*yaml.RNode, error) {
	// may require initializing the dest node
	dest, err := l.Sources.setDestNode(l.VisitList(l.Sources, AssociativeList))
	if dest == nil || err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "begin dest: %v\n", dest.MustString())

	// find the list of elements we need to recursively walk
	key, err := l.elementKey()
	if err != nil {
		return nil, err
	}
	values := l.elementValues(key)

	// recursively set the elements in the list
	for _, value := range values {
		// DEBUG: val is a combination of l.elementValue() rnodes
		val, err := Walker{Visitor: l,
			Sources: l.elementValue(key, value)}.Walk()
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "val: %v\n", val.MustString())
		if yaml.IsEmpty(val) {
			_, err = dest.Pipe(yaml.ElementSetter{Key: key, Value: value})
			if err != nil {
				return nil, err
			}
			continue
		}

		path, field := val.DeepField(key)
		if field == nil && len(path) == 1 {
			// make sure the key is set on the field
			_, err = val.Pipe(yaml.SetField(key, yaml.NewScalarRNode(value)))
			if err != nil {
				return nil, err
			}
		}

		// this handles empty and non-empty values
		switch len(path) {
		case 0, 1:
			_, err = dest.Pipe(yaml.ElementSetter{Element: val.YNode(), Key: key, Value: value})
			if err != nil {
				return nil, err
			}
		default:
			destElems, err := dest.Elements()
			if err != nil {
				return nil, err
			}
			destValue, err := destElems[0].Pipe(yaml.PathGetter{Path: path})
			fmt.Fprintf(os.Stderr, "destElem: %v\n", destElems[0].MustString())

			var elem *yaml.Node
			if destValue.Document().Value != value {
				elem = val.YNode()
				fmt.Fprintf(os.Stderr, "elem: %v\n", elem.Value)
			}
			_, err = dest.Pipe(yaml.ElementSetter{Element: elem, Key: key, Value: value})
			if err != nil {
				return nil, err
			}
		}
	}
	// field is empty
	if yaml.IsEmpty(dest) {
		return nil, nil
	}
	fmt.Fprintf(os.Stderr, "return dest: %v\n", dest.MustString())
	return dest, nil
}

// elementKey returns the merge key to use for the associative list
func (l Walker) elementKey() (string, error) {
	var key string
	for i := range l.Sources {
		if l.Sources[i] != nil && len(l.Sources[i].Content()) > 0 {
			newKey := l.Sources[i].GetAssociativeKey()
			if key != "" && key != newKey {
				return "", errors.Errorf(
					"conflicting merge keys [%s,%s] for field %s",
					key, newKey, strings.Join(l.Path, "."))
			}
			key = newKey
		}
	}
	if key == "" {
		return "", errors.Errorf("no merge key found for field %s",
			strings.Join(l.Path, "."))
	}
	return key, nil
}

// elementValues returns a slice containing all values for the field across all elements
// from all sources.
// Return value slice is ordered using the original ordering from the elements, where
// elements missing from earlier sources appear later.
func (l Walker) elementValues(key string) []string {
	// use slice to to keep elements in the original order
	// dest node must be first
	var returnValues []string
	seen := sets.String{}
	for i := range l.Sources {
		if l.Sources[i] == nil {
			continue
		}

		// add the value of the field for each element
		// don't check error, we know this is a list node
		values, _ := l.Sources[i].ElementValues(key)
		for _, s := range values {
			if seen.Has(s) {
				continue
			}
			returnValues = append(returnValues, s)
			seen.Insert(s)
		}
	}
	return returnValues
}

// fieldValue returns a slice containing each source's value for fieldName
func (l Walker) elementValue(key, value string) []*yaml.RNode {
	var fields []*yaml.RNode
	for i := range l.Sources {
		if l.Sources[i] == nil {
			fields = append(fields, nil)
			continue
		}
		fields = append(fields, l.Sources[i].Element(key, value))
	}
	return fields
}
