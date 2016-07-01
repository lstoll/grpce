package reporters

type ErrorReporter interface {
	ReportError(err error)
}

type MetricsReporter interface {
	Count(key string, by int64)
	Gauge(key string, val int64)
}

func ReportError(r ErrorReporter, err error) {
	if r != nil {
		r.ReportError(err)
	}
}

func ReportCount(r MetricsReporter, key string, by int64) {
	if r != nil {
		r.Count(key, by)
	}
}

func ReportGauge(r MetricsReporter, key string, val int64) {
	if r != nil {
		r.Gauge(key, val)
	}
}
