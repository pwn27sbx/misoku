package tutor

// Resume returns the lesson note to open for the player whose progress is p:
// the index at the start, once every lesson passes, and when there is no
// progress at all; otherwise the first lesson not passed yet, so a player
// who left a lesson half done comes back to it.
func Resume(p *Progress) string {
	cat := p.catalog()
	lessons := flatten(cat)
	if len(lessons) == 0 {
		return ""
	}
	index := lessons[0].File
	done, total := p.Count()
	if done == 0 || done == total || total == 0 {
		return index
	}
	for _, l := range lessons {
		if len(l.Steps) > 0 && !p.store.LessonDone(l) {
			return l.File
		}
	}
	return index
}
