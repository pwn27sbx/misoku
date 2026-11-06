package tutor

// Resume returns the lesson note to open for the player whose progress is p:
// the index at the start, once every level passes, and when there is no
// progress at all; otherwise the first lesson not passed yet, so a player
// who left level 4 half done comes back to it.
func Resume(p *Progress) string {
	done, total := p.Count()
	if done == 0 || done == total || total == 0 {
		return Lessons[0].File
	}
	for _, l := range Lessons {
		if l.Check != "" && !p.done[l.File] {
			return l.File
		}
	}
	return Lessons[0].File
}
