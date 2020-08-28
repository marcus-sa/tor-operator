package metrics

import (
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/yawning/bulb"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"strconv"
	"strings"
)

var (
	circuitsCountGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "circuits",
			Help: "Count of currently open circuits",
		},
	)
	streamsCountGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "streams",
			Help: "Count of currently open streams",
		},
	)
	orconnsCountGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "orconns",
			Help: "Count of currently open ORConns",
		},
	)
	trafficReadCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "traffic_read",
			Help: "Total traffic read",
		},
	)
	trafficWrittenCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "traffic_written",
			Help: "Total traffic written",
		},
	)
)

type TorDaemonMetricsExporter struct {
	Bulb *bulb.Conn
}

func (e *TorDaemonMetricsExporter) Connect(socket, port string) error  {
	var err error

	if socket != "" {
		e.Bulb, err = bulb.Dial("unix", socket)
	} else if port != "" {
		e.Bulb, err = bulb.Dial("tcp4", port)
	} else {
		return errors.New("no endpoint specified")
	}

	if err != nil {
		return errors.Wrap(err, "failed to connect to Tor")
	}

	//c.Debug(true)
	if err = e.Bulb.Authenticate(""); err != nil {
		return errors.Wrap(err, "Could not authenticate")
	}

	return nil
}

func (e* TorDaemonMetricsExporter) Reconnect(socket, port string) error {
	if err := e.Close(); err != nil {
		return err
	}

	return e.Connect(socket, port)
}

func (e *TorDaemonMetricsExporter) Close() error {
	defer e.Bulb.Close()
	return nil
}

func (e *TorDaemonMetricsExporter) getInfoFloat(val string) (float64, error) {
	resp, err := e.Bulb.Request("GETINFO " + val)
	if err != nil {
		return -1, err
	}

	if len(resp.Data) != 1 {
		return -1, errors.Errorf("GETINFO %s returned unknown response", val)
	}

	vals := strings.SplitN(resp.Data[0], "=", 2)
	if len(vals) != 2 {
		return -1, errors.Errorf("GETINFO %s returned invalid response", val)
	}

	return strconv.ParseFloat(vals[1], 64)
}

// Do a GETINFO %val, return the number of lines that match %match
func (e *TorDaemonMetricsExporter) linesWithMatch(val, match string) (int, error) {
	resp, err := e.Bulb.Request("GETINFO " + val)
	if err != nil {
		return -1, errors.Wrap(err, "GETINFO circuit-status failed")
	}
	if len(resp.Data) < 2 {
		return 0, nil
	}

	ct := 0
	for _, line := range strings.Split(resp.Data[1], "\n") {
		if strings.Contains(line, match) {
			ct++
		}
	}

	return ct, nil
}

func (e *TorDaemonMetricsExporter) scrapeTraffic() error {
	if trafficRead, err := e.getInfoFloat("traffic/read"); err != nil {
		return errors.Wrap(err, "could not scrape read_bytes")
	} else {
		trafficReadCounter.Add(trafficRead)
	}

	if trafficWritten, err := e.getInfoFloat("traffic/written"); err != nil {
		return errors.Wrap(err, "could not scrape written_bytes")
	} else {
		trafficWrittenCounter.Add(trafficWritten)
	}

	return nil
}

func (e *TorDaemonMetricsExporter) scrapeStatus() error {
	if circuitsCount, err := e.linesWithMatch("circuit-status", " BUILT "); err != nil {
		return errors.Wrapf(err, "could not scrape circuit-status")
	} else {
		circuitsCountGauge.Set(float64(circuitsCount))
	}

	if streamsCount, err := e.linesWithMatch("stream-status", "SUCCEEDED"); err != nil {
		return errors.Wrapf(err, "could not scrape stream-status")
	} else {
		streamsCountGauge.Set(float64(streamsCount))
	}

	if orconnsCount, err := e.linesWithMatch("orconn-status", " CONNECTED"); err != nil {
		return errors.Wrapf(err, "could not scrape orconn-status")
	} else {
		orconnsCountGauge.Set(float64(orconnsCount))
	}

	return nil
}

func (e *TorDaemonMetricsExporter) Start(socket, port string) error {
	if err := e.Connect(socket, port); err != nil {
		return err
	}

	metrics.Registry.MustRegister(trafficWrittenCounter, trafficReadCounter, orconnsCountGauge, streamsCountGauge, circuitsCountGauge)

	return nil
}
