package intent

type RecognitionInput struct {
	OriginalQuestion         string
	RewrittenQuestion        string
	SubQuestions             []string
	HistorySummary           string
	RecentQuestionTranscript string
}
