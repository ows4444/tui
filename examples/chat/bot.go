package main

// A chat needs someone to talk to, and this example is offline: each channel
// has one scripted person who answers in turn from a fixed list, so the same
// messages always get the same replies.

type message struct {
	from string
	at   string // clock time as shown, e.g. "09:41"
	text string
}

type channel struct {
	name    string
	who     string // the one person who answers here
	replies []string
	msgs    []message
	unread  int
}

func sampleChannels() []channel {
	return []channel{
		{
			name: "general", who: "mira",
			replies: []string{
				"Nice, thanks. I will pass that on to the rest of the team.",
				"Can you say more about that? I want to be sure I follow.",
				"That works for me. Shall we call it done and move on to the next thing on the list?",
				"Agreed. Write it down in the notes so nobody has to ask twice.",
			},
			msgs: []message{
				{"mira", "09:12", "Morning! The release branch is cut and the checks are green."},
				{"jon", "09:15", "Great. I will write the changelog after lunch."},
				{"mira", "09:16", "Please keep it short: one line per change, newest first."},
				{"you", "09:20", "On it. Do we mention the scroll region fix?"},
			},
		},
		{
			name: "design", who: "kai",
			replies: []string{
				"I tried it at 40 columns and the sidebar disappears, which is what I wanted.",
				"The accent colour is doing too much work. One highlight per screen is enough.",
				"Looks good to me. Ship it, and we can polish the spacing next week.",
			},
			msgs: []message{
				{"kai", "08:40", "New mock for the settings page is in the folder."},
				{"kai", "08:41", "Left list, right pane, toggles and sliders. Nothing fancy."},
			},
			unread: 2,
		},
		{
			name: "random", who: "bo",
			replies: []string{
				"Ha. That reminds me of a bug I once spent a whole day on.",
				"Coffee first, then I will look. No promises.",
				"Same here. The terminal is the best place to work, and I will not hear otherwise.",
			},
			msgs: []message{
				{"bo", "07:58", "Anyone else still using a 24-row terminal on purpose?"},
			},
			unread: 1,
		},
	}
}
