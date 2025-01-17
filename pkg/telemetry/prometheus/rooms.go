// Copyright 2023 LiveKit, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheus

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/atomic"

	"github.com/livekit/protocol/livekit"
)

var (
	roomCurrent            atomic.Int32
	participantCurrent     atomic.Int32
	trackPublishedCurrent  atomic.Int32
	trackSubscribedCurrent atomic.Int32
	trackPublishAttempts   atomic.Int32
	trackPublishSuccess    atomic.Int32
	trackSubscribeAttempts atomic.Int32
	trackSubscribeSuccess  atomic.Int32
	// count the number of failures that are due to user error (permissions, track doesn't exist), so we could compute
	// success rate by subtracting this from total attempts
	trackSubscribeUserError atomic.Int32

	promRoomCurrent            prometheus.Gauge
	promRoomDuration           prometheus.Histogram
	promParticipantCurrent     prometheus.Gauge
	promTrackPublishedCurrent  *prometheus.GaugeVec
	promTrackSubscribedCurrent *prometheus.GaugeVec
	promTrackPublishCounter    *prometheus.CounterVec
	promTrackSubscribeCounter  *prometheus.CounterVec
)

func initRoomStats(nodeID string, nodeType livekit.NodeType, env string) {
	promRoomCurrent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   livekitNamespace,
		Subsystem:   "room",
		Name:        "total",
		ConstLabels: prometheus.Labels{"node_id": nodeID, "node_type": nodeType.String(), "env": env},
	})
	promRoomDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace:   livekitNamespace,
		Subsystem:   "room",
		Name:        "duration_seconds",
		ConstLabels: prometheus.Labels{"node_id": nodeID, "node_type": nodeType.String(), "env": env},
		Buckets: []float64{
			5, 10, 60, 5 * 60, 10 * 60, 30 * 60, 60 * 60, 2 * 60 * 60, 5 * 60 * 60, 10 * 60 * 60,
		},
	})
	promParticipantCurrent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   livekitNamespace,
		Subsystem:   "participant",
		Name:        "total",
		ConstLabels: prometheus.Labels{"node_id": nodeID, "node_type": nodeType.String(), "env": env},
	})
	promTrackPublishedCurrent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   livekitNamespace,
		Subsystem:   "track",
		Name:        "published_total",
		ConstLabels: prometheus.Labels{"node_id": nodeID, "node_type": nodeType.String(), "env": env},
	}, []string{"kind"})
	promTrackSubscribedCurrent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   livekitNamespace,
		Subsystem:   "track",
		Name:        "subscribed_total",
		ConstLabels: prometheus.Labels{"node_id": nodeID, "node_type": nodeType.String(), "env": env},
	}, []string{"kind"})
	promTrackPublishCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   livekitNamespace,
		Subsystem:   "track",
		Name:        "publish_counter",
		ConstLabels: prometheus.Labels{"node_id": nodeID, "node_type": nodeType.String(), "env": env},
	}, []string{"kind", "state"})
	promTrackSubscribeCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   livekitNamespace,
		Subsystem:   "track",
		Name:        "subscribe_counter",
		ConstLabels: prometheus.Labels{"node_id": nodeID, "node_type": nodeType.String(), "env": env},
	}, []string{"state", "error"})

	prometheus.MustRegister(promRoomCurrent)
	prometheus.MustRegister(promRoomDuration)
	prometheus.MustRegister(promParticipantCurrent)
	prometheus.MustRegister(promTrackPublishedCurrent)
	prometheus.MustRegister(promTrackSubscribedCurrent)
	prometheus.MustRegister(promTrackPublishCounter)
	prometheus.MustRegister(promTrackSubscribeCounter)
}

func RoomStarted() {
	promRoomCurrent.Add(1)
	roomCurrent.Inc()
}

func RoomEnded(startedAt time.Time) {
	if !startedAt.IsZero() {
		promRoomDuration.Observe(float64(time.Since(startedAt)) / float64(time.Second))
	}
	promRoomCurrent.Sub(1)
	roomCurrent.Dec()
}

func AddParticipant() {
	promParticipantCurrent.Add(1)
	participantCurrent.Inc()
}

func SubParticipant() {
	promParticipantCurrent.Sub(1)
	participantCurrent.Dec()
}

func AddPublishedTrack(kind string) {
	promTrackPublishedCurrent.WithLabelValues(kind).Add(1)
	trackPublishedCurrent.Inc()
}

func SubPublishedTrack(kind string) {
	promTrackPublishedCurrent.WithLabelValues(kind).Sub(1)
	trackPublishedCurrent.Dec()
}

func AddPublishAttempt(kind string) {
	trackPublishAttempts.Inc()
	promTrackPublishCounter.WithLabelValues(kind, "attempt").Inc()
}

func AddPublishSuccess(kind string) {
	trackPublishSuccess.Inc()
	promTrackPublishCounter.WithLabelValues(kind, "success").Inc()
}

func RecordTrackSubscribeSuccess(kind string) {
	// modify both current and total counters
	promTrackSubscribedCurrent.WithLabelValues(kind).Add(1)
	trackSubscribedCurrent.Inc()

	promTrackSubscribeCounter.WithLabelValues("success", "").Inc()
	trackSubscribeSuccess.Inc()
}

func RecordTrackUnsubscribed(kind string) {
	// unsubscribed modifies current counter, but we leave the total values alone since they
	// are used to compute rate
	promTrackSubscribedCurrent.WithLabelValues(kind).Sub(1)
	trackSubscribedCurrent.Dec()
}

func RecordTrackSubscribeAttempt() {
	trackSubscribeAttempts.Inc()
	promTrackSubscribeCounter.WithLabelValues("attempt", "").Inc()
}

func RecordTrackSubscribeFailure(err error, isUserError bool) {
	promTrackSubscribeCounter.WithLabelValues("failure", err.Error()).Inc()

	if isUserError {
		trackSubscribeUserError.Inc()
	}
}
