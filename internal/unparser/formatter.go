/*
Copyright 2019 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package formatter provides API for producing pretty-printed source
// from AST.
package unparser

import (
	"github.com/google/go-jsonnet/ast"
)

// StringStyle controls how the reformatter rewrites string literals.
// Strings that contain a ' or a " use the optimal syntax to avoid escaping
// those characters.
type StringStyle int

const (
	// StringStyleDouble means "this".
	StringStyleDouble StringStyle = iota
	// StringStyleSingle means 'this'.
	StringStyleSingle
	// StringStyleLeave means strings are left how they were found.
	StringStyleLeave
)

// CommentStyle controls how the reformatter rewrites comments.
// Comments that look like a #! hashbang are always left alone.
type CommentStyle int

const (
	// CommentStyleHash means #.
	CommentStyleHash CommentStyle = iota
	// CommentStyleSlash means //.
	CommentStyleSlash
	// CommentStyleLeave means comments are left as they are found.
	CommentStyleLeave
)

// Options is a set of parameters that control the reformatter's behaviour.
type Options struct {
	// Indent is the number of spaces for each level of indenation.
	Indent int
	// MaxBlankLines is the max allowed number of consecutive blank lines.
	MaxBlankLines int
	StringStyle   StringStyle
	CommentStyle  CommentStyle
	// PrettyFieldNames causes fields to only be wrapped in '' when needed.
	PrettyFieldNames bool
	// PadArrays causes arrays to be written like [ this ] instead of [this].
	PadArrays bool
	// PadObjects causes arrays to be written like { this } instead of {this}.
	PadObjects bool
	// SortImports causes imports at the top of the file to be sorted in groups
	// by filename.
	SortImports bool
	// UseImplicitPlus removes plus sign where it is not required.
	UseImplicitPlus bool

	StripEverything     bool
	StripComments       bool
	StripAllButComments bool
}

// DefaultOptions returns the recommended formatter behaviour.
func DefaultOptions() Options {
	return Options{
		Indent:           2,
		MaxBlankLines:    2,
		StringStyle:      StringStyleSingle,
		CommentStyle:     CommentStyleSlash,
		UseImplicitPlus:  true,
		PrettyFieldNames: true,
		PadArrays:        false,
		PadObjects:       true,
		SortImports:      true,
	}
}

// If left recursive, return the left hand side, else return nullptr.
func leftRecursive(expr ast.Node) ast.Node {
	switch node := expr.(type) {
	case *ast.Apply:
		return node.Target
	case *ast.ApplyBrace:
		return node.Left
	case *ast.Binary:
		return node.Left
	case *ast.Index:
		return node.Target
	case *ast.InSuper:
		return node.Index
	case *ast.Slice:
		return node.Target
	default:
		return nil
	}
}

// leftRecursiveDeep is the transitive closure of leftRecursive.
// It only returns nil when called with nil.
func leftRecursiveDeep(expr ast.Node) ast.Node {
	last := expr
	left := leftRecursive(expr)
	for left != nil {
		last = left
		left = leftRecursive(last)
	}
	return last
}

func openFodder(node ast.Node) *ast.Fodder {
	return leftRecursiveDeep(node).OpenFodder()
}

func removeInitialNewlines(node ast.Node) {
	f := openFodder(node)
	for len(*f) > 0 && (*f)[0].Kind == ast.FodderLineEnd {
		*f = (*f)[1:]
	}
}

func removeExtraTrailingNewlines(finalFodder ast.Fodder) {
	if len(finalFodder) > 0 {
		finalFodder[len(finalFodder)-1].Blanks = 0
	}
}
