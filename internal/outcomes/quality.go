package outcomes

import (
	"strings"
	"time"
)

type QualityClassification string

const (
	QualityClassNormal            QualityClassification = "normal"
	QualityClassPrivacyProxy      QualityClassification = "privacy_proxy"
	QualityClassLinkScanner       QualityClassification = "link_scanner"
	QualityClassDuplicateCallback QualityClassification = "duplicate_callback"
	QualityClassLateEvent         QualityClassification = "late_event"
)

type QualityTag struct {
	Classification QualityClassification `json:"classification"`
	RuleID         string                `json:"rule_id"`
	Reason         string                `json:"reason"`
	IsFiltered     bool                  `json:"is_filtered"`
}

type QualityClassifier struct {
	LateWatermarkCutoff time.Duration
}

func NewQualityClassifier() *QualityClassifier {
	return &QualityClassifier{
		LateWatermarkCutoff: 30 * 24 * time.Hour,
	}
}

func (qc *QualityClassifier) Classify(event *OutcomeEvent, isDuplicate bool) QualityTag {
	if isDuplicate {
		return QualityTag{
			Classification: QualityClassDuplicateCallback,
			RuleID:         "RULE_DUPLICATE_CALLBACK",
			Reason:         "Duplicate callback event ID or source identity ingested",
			IsFiltered:     true,
		}
	}

	normEventType := strings.ToLower(strings.TrimSpace(event.EventType))
	if normEventType == "open" || normEventType == "open_event" {
		if qc.isPrivacyProxy(event) {
			return QualityTag{
				Classification: QualityClassPrivacyProxy,
				RuleID:         "RULE_PRIVACY_PROXY",
				Reason:         "Apple Mail Privacy Protection or privacy image proxy detected",
				IsFiltered:     true,
			}
		}
	}

	if normEventType == "click" || normEventType == "click_event" {
		if qc.isLinkScanner(event) {
			return QualityTag{
				Classification: QualityClassLinkScanner,
				RuleID:         "RULE_LINK_SCANNER",
				Reason:         "Security link scanner or bot crawler detected",
				IsFiltered:     true,
			}
		}
	}

	if qc.isLateEvent(event) {
		return QualityTag{
			Classification: QualityClassLateEvent,
			RuleID:         "RULE_LATE_EVENT",
			Reason:         "Event timestamp exceeds watermark cutoff window",
			IsFiltered:     true,
		}
	}

	return QualityTag{
		Classification: QualityClassNormal,
		RuleID:         "RULE_NORMAL",
		Reason:         "Valid first-party tracking or callback event",
		IsFiltered:     false,
	}
}

func (qc *QualityClassifier) isPrivacyProxy(event *OutcomeEvent) bool {
	ua := strings.ToLower(event.UserAgent)
	proxySignatures := []string{
		"apple mail privacy protection",
		"googleimageproxy",
		"yahoomailproxy",
		"cloudflare-proxy",
		"gmailproxy",
		"applemail",
	}
	for _, sig := range proxySignatures {
		if strings.Contains(ua, sig) {
			return true
		}
	}
	if event.Metadata != nil {
		if val, ok := event.Metadata["privacy_proxy"].(bool); ok && val {
			return true
		}
		if val, ok := event.Metadata["is_privacy_proxy"].(bool); ok && val {
			return true
		}
	}
	return false
}

func (qc *QualityClassifier) isLinkScanner(event *OutcomeEvent) bool {
	ua := strings.ToLower(event.UserAgent)
	botKeywords := []string{
		"bot", "crawler", "spider", "barracuda", "proofpoint",
		"mimecast", "scanner", "googlebot", "bingbot", "virus",
		"security", "linkchecker", "automated",
	}
	for _, kw := range botKeywords {
		if strings.Contains(ua, kw) {
			return true
		}
	}
	if event.Metadata != nil {
		if val, ok := event.Metadata["link_scanner"].(bool); ok && val {
			return true
		}
		if val, ok := event.Metadata["is_bot"].(bool); ok && val {
			return true
		}
	}
	return false
}

func (qc *QualityClassifier) isLateEvent(event *OutcomeEvent) bool {
	if event.Metadata != nil {
		if val, ok := event.Metadata["is_late"].(bool); ok && val {
			return true
		}
	}
	if !event.Timestamp.IsZero() && qc.LateWatermarkCutoff > 0 {
		if time.Since(event.Timestamp) > qc.LateWatermarkCutoff {
			return true
		}
	}
	return false
}
