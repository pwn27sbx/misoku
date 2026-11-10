package tutor

// ResumeNote returns the note to open for a tutorial session on route: the
// route's first lesson with an unfinished step, or its index when every step
// is done. An explicit route on a route nobody started opens its index (Soy
// nuevo, Ya tengo experiencia); without one, the last route played or the
// first route resumes where the player left.
func ResumeNote(cat Catalog, s *Store, route string) string {
	r, ok := cat.RouteByID(route)
	explicit := route != ""
	if !ok {
		if last := s.LastRoute(); last != "" {
			r, ok = cat.RouteByID(last)
		}
		if !ok {
			r, ok = cat.FirstRoute()
		}
	}
	if !ok {
		return ""
	}
	if explicit && !s.RouteStarted(r) {
		return r.IndexPath()
	}
	for _, l := range r.Lessons {
		if s.NextStep(l) != 0 {
			return l.Path()
		}
	}
	return r.IndexPath()
}
