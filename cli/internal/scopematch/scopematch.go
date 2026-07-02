package scopematch

import (
	"path/filepath"
	"strings"
)

func MatchesAllowed(relFile, allowed string) bool {
	file := filepath.Clean(relFile)
	if strings.Contains(allowed, "*") {
		if len(allowed) > 4096 {
			return false
		}
		return globstarMatch(strings.Split(file, "/"), strings.Split(allowed, "/"))
	}
	normalizedAllowed := filepath.Clean(allowed)
	return file == normalizedAllowed || strings.HasPrefix(file, normalizedAllowed+string(filepath.Separator))
}

func segmentMatch(seg, pat string) bool {
	s, p := 0, 0
	starIdx := -1
	matchIdx := 0
	for s < len(seg) {
		if p < len(pat) && (pat[p] == '?' || pat[p] == seg[s]) {
			s++
			p++
		} else if p < len(pat) && pat[p] == '*' {
			starIdx = p
			matchIdx = s
			p++
		} else if starIdx != -1 {
			p = starIdx + 1
			matchIdx++
			s = matchIdx
		} else {
			return false
		}
	}
	for p < len(pat) && pat[p] == '*' {
		p++
	}
	return p == len(pat)
}

func globstarMatch(fileSegs, laneSegs []string) bool {
	f, l := 0, 0
	starIdx := -1
	matchIdx := 0
	for f < len(fileSegs) {
		if l < len(laneSegs) && laneSegs[l] == "**" {
			starIdx = l
			matchIdx = f
			l++
		} else if l < len(laneSegs) && segmentMatch(fileSegs[f], laneSegs[l]) {
			f++
			l++
		} else if starIdx != -1 {
			l = starIdx + 1
			matchIdx++
			f = matchIdx
		} else {
			return false
		}
	}
	for l < len(laneSegs) && laneSegs[l] == "**" {
		l++
	}
	return l == len(laneSegs)
}
