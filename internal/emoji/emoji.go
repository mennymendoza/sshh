package emoji

import (
	"sort"
	"strings"
)

type Entry struct {
	Char        string
	Description string
}

var catalog = []Entry{
	{"😀", "grin"},
	{"😄", "happy"},
	{"😂", "joy"},
	{"🤣", "rofl"},
	{"🙂", "smile"},
	{"😉", "wink"},
	{"😊", "blush"},
	{"😍", "loveeyes"},
	{"🤩", "starstruck"},
	{"😘", "kiss"},
	{"😋", "yum"},
	{"😎", "cool"},
	{"🥳", "partyface"},
	{"😏", "smirk"},
	{"😒", "unamused"},
	{"😢", "cry"},
	{"😭", "sob"},
	{"😡", "angry"},
	{"🤯", "mindblown"},
	{"😳", "flushed"},
	{"😱", "scream"},
	{"🤔", "thinking"},
	{"🙄", "eyeroll"},
	{"😴", "sleepy"},
	{"🤢", "sick"},
	{"🤠", "cowboy"},
	{"🤡", "clown"},
	{"👻", "ghost"},
	{"💀", "skull"},
	{"🤖", "robot"},
	{"❤️", "heart"},
	{"🧡", "orangeheart"},
	{"💛", "yellowheart"},
	{"💚", "greenheart"},
	{"💙", "blueheart"},
	{"💜", "purpleheart"},
	{"💔", "brokenheart"},
	{"💕", "twohearts"},
	{"👍", "thumbsup"},
	{"👎", "thumbsdown"},
	{"👌", "ok"},
	{"✌️", "peace"},
	{"🤞", "fingerscrossed"},
	{"👏", "clap"},
	{"🙌", "raisedhands"},
	{"👋", "wave"},
	{"🤝", "handshake"},
	{"🙏", "pray"},
	{"💪", "muscle"},
	{"✊", "fist"},
	{"👊", "fistbump"},
	{"🖕", "middlefinger"},
	{"🔥", "fire"},
	{"💯", "hundred"},
	{"✨", "sparkles"},
	{"⭐", "star"},
	{"💥", "boom"},
	{"🎉", "confetti"},
	{"🎁", "gift"},
	{"🏆", "trophy"},
	{"✅", "check"},
	{"❌", "cross"},
	{"❓", "question"},
	{"❗", "exclamation"},
	{"⚡", "lightning"},
	{"☕", "coffee"},
	{"🍕", "pizza"},
	{"🍺", "beer"},
	{"🚀", "rocket"},
	{"💡", "idea"},
}

func Search(query string) []Entry {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}

	type match struct {
		entry Entry
		score int
	}

	var matches []match
	for _, e := range catalog {
		if s := fuzzyScore(query, strings.ToLower(e.Description)); s > 0 {
			matches = append(matches, match{e, s})
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	results := make([]Entry, len(matches))
	for i, m := range matches {
		results[i] = m.entry
	}
	return results
}

func fuzzyScore(query, target string) int {
	qi := 0
	score := 0
	consecutive := 0
	for ti := 0; ti < len(target) && qi < len(query); ti++ {
		if target[ti] == query[qi] {
			score++
			if consecutive > 0 {
				score += 2
			}
			consecutive++
			qi++
		} else {
			consecutive = 0
		}
	}
	if qi < len(query) {
		return 0
	}

	switch {
	case target == query:
		score += 100
	case strings.HasPrefix(target, query):
		score += 50
	}
	return score
}
