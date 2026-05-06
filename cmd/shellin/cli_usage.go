// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"flag"
	"fmt"
	"strings"
)

func setFilteredUsage(fs *flag.FlagSet, hidden map[string]bool) {
	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintf(out, "Usage of %s:\n", fs.Name())
		fs.VisitAll(func(f *flag.Flag) {
			if hidden[f.Name] {
				return
			}
			name, usage := flag.UnquoteUsage(f)
			if name == "" {
				fmt.Fprintf(out, "  -%s\n", f.Name)
			} else {
				fmt.Fprintf(out, "  -%s %s\n", f.Name, name)
			}

			defaultText := ""
			if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
				if f.DefValue == "true" {
					defaultText = " (default true)"
				}
			} else if f.DefValue != "" {
				if strings.ContainsAny(f.DefValue, " \t") {
					defaultText = fmt.Sprintf(" (default %q)", f.DefValue)
				} else {
					defaultText = fmt.Sprintf(" (default %s)", f.DefValue)
				}
			}
			fmt.Fprintf(out, "    \t%s%s\n", usage, defaultText)
		})
	}
}
