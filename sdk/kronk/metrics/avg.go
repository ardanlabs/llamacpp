package metrics

import "expvar"

type avgMetric struct {
	sum   *expvar.Int
	count *expvar.Int
	min   *expvar.Int
	max   *expvar.Int
}

func newAvgMetric(name string) *avgMetric {
	a := &avgMetric{
		sum:   expvar.NewInt(name + "_sum"),
		count: expvar.NewInt(name + "_count"),
		min:   expvar.NewInt(name + "_min"),
		max:   expvar.NewInt(name + "_max"),
	}

	expvar.Publish(name+"_avg", expvar.Func(func() any {
		return a.average()
	}))

	return a
}

func (a *avgMetric) add(value int64) {
	a.sum.Add(value)
	a.count.Add(1)

	if a.count.Value() == 1 || value < a.min.Value() {
		a.min.Set(value)
	}

	if value > a.max.Value() {
		a.max.Set(value)
	}
}

func (a *avgMetric) average() int64 {
	c := a.count.Value()
	if c == 0 {
		return 0
	}

	return a.sum.Value() / c
}
