package i18n

// Messages of the tutorial inside the app.
const (
	TutorSaveFailed Key = "tutor.save_failed"
)

func init() {
	register(EN, map[Key]string{
		TutorSaveFailed: "Could not save the tutorial's progress: %s",
	})
	register(ES, map[Key]string{
		TutorSaveFailed: "No se pudo guardar el avance del tutorial: %s",
	})
}
