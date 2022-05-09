package telemetry

import (
	"context"
	"time"

	"github.com/livekit/livekit-server/pkg/config"
	"github.com/livekit/livekit-server/pkg/utils"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/logger"
	"github.com/livekit/protocol/webhook"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . TelemetryService
type TelemetryService interface {
	// stats
	TrackStats(streamType livekit.StreamType, participantID livekit.ParticipantID, trackID livekit.TrackID, stat *livekit.AnalyticsStat)

	// events
	RoomStarted(ctx context.Context, room *livekit.Room)
	RoomEnded(ctx context.Context, room *livekit.Room)
	ParticipantJoined(ctx context.Context, room *livekit.Room, participant *livekit.ParticipantInfo, clientInfo *livekit.ClientInfo, clientMeta *livekit.AnalyticsClientMeta)
	ParticipantActive(ctx context.Context, room *livekit.Room, participant *livekit.ParticipantInfo, clientMeta *livekit.AnalyticsClientMeta)
	ParticipantLeft(ctx context.Context, room *livekit.Room, participant *livekit.ParticipantInfo)
	TrackPublished(ctx context.Context, participantID livekit.ParticipantID, identity livekit.ParticipantIdentity, track *livekit.TrackInfo)
	TrackUnpublished(ctx context.Context, participantID livekit.ParticipantID, identity livekit.ParticipantIdentity, track *livekit.TrackInfo, ssrc uint32)
	TrackSubscribed(ctx context.Context, participantID livekit.ParticipantID, track *livekit.TrackInfo, publisher *livekit.ParticipantInfo)
	TrackUnsubscribed(ctx context.Context, participantID livekit.ParticipantID, track *livekit.TrackInfo)
	TrackPublishedUpdate(ctx context.Context, participantID livekit.ParticipantID, track *livekit.TrackInfo)
	TrackMaxSubscribedVideoQuality(ctx context.Context, participantID livekit.ParticipantID, track *livekit.TrackInfo, mime string, maxQuality livekit.VideoQuality)
	RecordingStarted(ctx context.Context, ri *livekit.RecordingInfo)
	RecordingEnded(ctx context.Context, ri *livekit.RecordingInfo)
	EgressStarted(ctx context.Context, info *livekit.EgressInfo)
	EgressEnded(ctx context.Context, info *livekit.EgressInfo)
}

type telemetryService struct {
	internalService TelemetryServiceInternal
	jobQueue        *utils.OpsQueue
}

// queue should be sufficiently large to avoid blocking
const jobQueueBufferSize = 10000

func NewTelemetryService(notifier webhook.Notifier, analytics AnalyticsService) TelemetryService {
	t := &telemetryService{
		internalService: NewTelemetryServiceInternal(notifier, analytics),
		jobQueue:        utils.NewOpsQueue(logger.GetDefaultLogger(), "telemetry", jobQueueBufferSize),
	}

	t.jobQueue.Start()

	go t.run()

	return t
}

func (t *telemetryService) run() {
	ticker := time.NewTicker(config.StatsUpdateInterval)
	defer ticker.Stop()
	for {
		<-ticker.C
		t.internalService.SendAnalytics()
	}
}

func (t *telemetryService) TrackStats(streamType livekit.StreamType, participantID livekit.ParticipantID, trackID livekit.TrackID, stats *livekit.AnalyticsStat) {
	t.jobQueue.Enqueue(func() {
		t.internalService.TrackStats(streamType, participantID, trackID, stats)
	})
}

func (t *telemetryService) RoomStarted(ctx context.Context, room *livekit.Room) {
	t.jobQueue.Enqueue(func() {
		t.internalService.RoomStarted(ctx, room)
	})
}

func (t *telemetryService) RoomEnded(ctx context.Context, room *livekit.Room) {
	t.jobQueue.Enqueue(func() {
		t.internalService.RoomEnded(ctx, room)
	})
}

func (t *telemetryService) ParticipantJoined(ctx context.Context, room *livekit.Room, participant *livekit.ParticipantInfo,
	clientInfo *livekit.ClientInfo, clientMeta *livekit.AnalyticsClientMeta) {
	t.jobQueue.Enqueue(func() {
		t.internalService.ParticipantJoined(ctx, room, participant, clientInfo, clientMeta)
	})
}

func (t *telemetryService) ParticipantActive(ctx context.Context, room *livekit.Room, participant *livekit.ParticipantInfo, clientMeta *livekit.AnalyticsClientMeta) {
	t.jobQueue.Enqueue(func() {
		t.internalService.ParticipantActive(ctx, room, participant, clientMeta)
	})
}

func (t *telemetryService) ParticipantLeft(ctx context.Context, room *livekit.Room, participant *livekit.ParticipantInfo) {
	t.jobQueue.Enqueue(func() {
		t.internalService.ParticipantLeft(ctx, room, participant)
	})
}

func (t *telemetryService) TrackPublished(ctx context.Context, participantID livekit.ParticipantID, identity livekit.ParticipantIdentity, track *livekit.TrackInfo) {
	t.jobQueue.Enqueue(func() {
		t.internalService.TrackPublished(ctx, participantID, identity, track)
	})
}

func (t *telemetryService) TrackUnpublished(ctx context.Context, participantID livekit.ParticipantID, identity livekit.ParticipantIdentity, track *livekit.TrackInfo, ssrc uint32) {
	t.jobQueue.Enqueue(func() {
		t.internalService.TrackUnpublished(ctx, participantID, identity, track, ssrc)
	})
}

func (t *telemetryService) TrackSubscribed(ctx context.Context, participantID livekit.ParticipantID, track *livekit.TrackInfo, publisher *livekit.ParticipantInfo) {
	t.jobQueue.Enqueue(func() {
		t.internalService.TrackSubscribed(ctx, participantID, track, publisher)
	})
}

func (t *telemetryService) TrackUnsubscribed(ctx context.Context, participantID livekit.ParticipantID, track *livekit.TrackInfo) {
	t.jobQueue.Enqueue(func() {
		t.internalService.TrackUnsubscribed(ctx, participantID, track)
	})
}

func (t *telemetryService) RecordingStarted(ctx context.Context, ri *livekit.RecordingInfo) {
	t.jobQueue.Enqueue(func() {
		t.internalService.RecordingStarted(ctx, ri)
	})
}

func (t *telemetryService) RecordingEnded(ctx context.Context, ri *livekit.RecordingInfo) {
	t.jobQueue.Enqueue(func() {
		t.internalService.RecordingEnded(ctx, ri)
	})
}

func (t *telemetryService) TrackPublishedUpdate(ctx context.Context, participantID livekit.ParticipantID, track *livekit.TrackInfo) {
	t.jobQueue.Enqueue(func() {
		t.internalService.TrackPublishedUpdate(ctx, participantID, track)
	})
}

func (t *telemetryService) TrackMaxSubscribedVideoQuality(ctx context.Context, participantID livekit.ParticipantID, track *livekit.TrackInfo, mime string, maxQuality livekit.VideoQuality) {
	t.jobQueue.Enqueue(func() {
		t.internalService.TrackMaxSubscribedVideoQuality(ctx, participantID, track, mime, maxQuality)
	})
}

func (t *telemetryService) EgressStarted(ctx context.Context, info *livekit.EgressInfo) {
	t.jobQueue.Enqueue(func() {
		t.internalService.EgressStarted(ctx, info)
	})
}

func (t *telemetryService) EgressEnded(ctx context.Context, info *livekit.EgressInfo) {
	t.jobQueue.Enqueue(func() {
		t.internalService.EgressEnded(ctx, info)
	})
}
