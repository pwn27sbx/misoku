package i18n

// Messages of the tutorial: the bar inside a lesson, the home screen's
// Learn entries, the first-run dialog and the installer's notice.
const (
	TutorSaveFailed Key = "tutor.save_failed"

	// TutorStepFlash is the flash of a step just passed (its number).
	TutorStepFlash Key = "tutor.step_flash"
	// TutorStepLine is the bar in a lesson: its ID, the step, the step
	// count and the step's text.
	TutorStepLine Key = "tutor.step_line"
	// TutorLessonPassed is the bar after a lesson: its ID and the next one.
	TutorLessonPassed Key = "tutor.lesson_passed"
	// TutorRouteDone is the bar after a route's last lesson (its name).
	TutorRouteDone Key = "tutor.route_done"
	// TutorAfterTutorial is the reference note's title, the "next" of the
	// last lesson.
	TutorAfterTutorial Key = "tutor.after_tutorial"
	// TutorNeedsGit is the bar of a lesson whose every step waits for git.
	TutorNeedsGit Key = "tutor.needs_git"
	// TutorStartFailed and TutorFailed report a tutorial run that did not
	// start or that failed.
	TutorStartFailed Key = "tutor.start_failed"
	TutorFailed      Key = "tutor.failed"
	// TutorLearnEntry is a route on the home screen: its name, done, total.
	TutorLearnEntry Key = "tutor.learn_entry"
	// TutorMigrated and TutorMigratedNoTrash say the old lesson notes were
	// retired, to the Trash or for good where the system has none.
	TutorMigrated        Key = "tutor.migrated"
	TutorMigratedNoTrash Key = "tutor.migrated_no_trash"

	// The first-run dialog.
	WelcomeTagline         Key = "welcome.tagline"
	WelcomeTaglineVim      Key = "welcome.tagline_vim"
	WelcomeQuestion        Key = "welcome.question"
	WelcomeQuestionCompact Key = "welcome.question_compact"
	WelcomeHint            Key = "welcome.hint"
	WelcomeNew             Key = "welcome.new"
	// WelcomeNewRoute is a plural: the Basic route's name and its lessons.
	WelcomeNewRoute         Key = "welcome.new_route"
	WelcomeExperienced      Key = "welcome.experienced"
	WelcomeExperiencedRoute Key = "welcome.experienced_route"
	WelcomeLater            Key = "welcome.later"
	WelcomeLaterDesc        Key = "welcome.later_desc"
)

func init() {
	register(EN, map[Key]string{
		TutorSaveFailed:      "Could not save the tutorial's progress: %s",
		TutorStepFlash:       "✓ Step %d",
		TutorStepLine:        "%s · Step %d/%d · %s",
		TutorLessonPassed:    "✓ %s passed · next: %s",
		TutorRouteDone:       "🏆 %s route complete",
		TutorAfterTutorial:   "After the tutorial",
		TutorNeedsGit:        "%s · You need git for this lesson",
		TutorStartFailed:     "Could not start the tutorial: %s",
		TutorFailed:          "Tutorial: %s",
		TutorLearnEntry:      "Learn · %s %d/%d",
		TutorMigrated:        "Tutorial updated: the previous tutorial's notes are in the Trash.",
		TutorMigratedNoTrash: "Tutorial updated: the previous tutorial's notes were deleted (this system has no Trash).",

		WelcomeTagline:          "Your notes in Markdown, in the terminal,",
		WelcomeTaglineVim:       "the Vim way.",
		WelcomeQuestion:         "Is this your first time here?",
		WelcomeQuestionCompact:  "First time in Misoku?",
		WelcomeHint:             "j/k move · enter choose",
		WelcomeNew:              "I'm new",
		WelcomeNewRoute + one:   "%s route · %d lesson",
		WelcomeNewRoute + other: "%s route · %d lessons",
		WelcomeExperienced:      "I have experience",
		WelcomeExperiencedRoute: "%s route",
		WelcomeLater:            "Not now",
		WelcomeLaterDesc:        "Straight to your notes",
	})
	register(ES, map[Key]string{
		TutorSaveFailed:      "No se pudo guardar el avance del tutorial: %s",
		TutorStepFlash:       "✓ Paso %d",
		TutorStepLine:        "%s · Paso %d/%d · %s",
		TutorLessonPassed:    "✓ %s superado · siguiente: %s",
		TutorRouteDone:       "🏆 Ruta %s completa",
		TutorAfterTutorial:   "Después del tutorial",
		TutorNeedsGit:        "%s · Necesitas git para esta lección",
		TutorStartFailed:     "No se pudo iniciar el tutorial: %s",
		TutorFailed:          "Tutorial: %s",
		TutorLearnEntry:      "Learn · %s %d/%d",
		TutorMigrated:        "Tutorial actualizado: las notas del tutorial anterior están en la papelera.",
		TutorMigratedNoTrash: "Tutorial actualizado: las notas del tutorial anterior se borraron (este sistema no tiene papelera).",

		WelcomeTagline:          "Tus notas en Markdown, en la terminal,",
		WelcomeTaglineVim:       "al estilo de Vim.",
		WelcomeQuestion:         "¿Es tu primera vez aquí?",
		WelcomeQuestionCompact:  "¿Es tu primera vez en Misoku?",
		WelcomeHint:             "j/k mover · enter elegir",
		WelcomeNew:              "Soy nuevo",
		WelcomeNewRoute + one:   "Ruta %s · %d lección",
		WelcomeNewRoute + other: "Ruta %s · %d lecciones",
		WelcomeExperienced:      "Ya tengo experiencia",
		WelcomeExperiencedRoute: "Ruta %s",
		WelcomeLater:            "Ahora no",
		WelcomeLaterDesc:        "Vas directo a tus notas",
	})
}
