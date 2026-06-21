package logs

import (
	"log"
	"strconv"
	"sync"
	"time"

	"sharelink/internal/config"
	"sharelink/internal/db"

	"gorm.io/gorm"
)

var (
	visitLogChan chan *db.VisitLog
	once         sync.Once
)

func Init() {
	once.Do(func() {
		visitLogChan = make(chan *db.VisitLog, 10000)
		go startLogWorker()
		go startCleanupWorker()
	})
}

// QueueLog pushes a visit log into the asynchronous queue
func QueueLog(vlog *db.VisitLog) {
	if visitLogChan == nil {
		return
	}

	select {
	case visitLogChan <- vlog:
	default:
		log.Println("WARNING: visit log queue is full, dropping log entry")
	}
}

func startLogWorker() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var batch []*db.VisitLog
	const maxBatchSize = 100

	flush := func() {
		if len(batch) == 0 {
			return
		}
		// Save batch inside transaction
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			return tx.Create(&batch).Error
		})
		if err != nil {
			log.Printf("failed to flush visit logs to database: %v", err)
		}
		batch = make([]*db.VisitLog, 0, maxBatchSize)
	}

	for {
		select {
		case vlog, ok := <-visitLogChan:
			if !ok {
				flush()
				return
			}
			batch = append(batch, vlog)
			if len(batch) >= maxBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func startCleanupWorker() {
	// Check every 12 hours
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	// Run once on startup
	runCleanup()

	for range ticker.C {
		runCleanup()
	}
}

func runCleanup() {
	// Check if cleanup is enabled
	enabledSetting, found, err := db.FindGlobalSetting("log_cleanup_enabled")
	if err == nil && found && (enabledSetting.Value == "false" || enabledSetting.Value == "0") {
		return // disabled
	}

	// Get retention days
	retentionDays := 90
	daysSetting, found, err := db.FindGlobalSetting("log_retention_days")
	if err == nil && found {
		if val, err := strconv.Atoi(daysSetting.Value); err == nil {
			retentionDays = val
		}
	}

	if retentionDays <= 0 {
		return // invalid value or infinite retention
	}

	threshold := config.NowUTC().AddDate(0, 0, -retentionDays)
	log.Printf("Starting automatic log cleanup. Deleting visit logs older than %d days (%s)...", retentionDays, threshold.Format("2006-01-02"))

	result := db.DB.Delete(&db.VisitLog{}, "access_time < ?", threshold)
	if result.Error != nil {
		log.Printf("failed to run automatic log cleanup: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("Automatic log cleanup deleted %d expired log entries", result.RowsAffected)
	}
}
