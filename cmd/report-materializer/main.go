package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("Initializing Report Materializer analytics service...")
	log.Println("tenant_id,experiment_id,experiment_version,journey_id,journey_version,mode,metric_name,denominator_name,variant_id,is_control,assigned,exposed,attempted,accepted,delivered,unique_open,unique_click,conversion,sample_size,absolute_rate,rate_ci_lower,rate_ci_upper,absolute_lift,lift_ci_lower,lift_ci_upper,relative_lift,rel_lift_ci_lower,rel_lift_ci_upper,p_value,is_stat_sig,srm_status,srm_p_value,watermark,window_state")
	log.Println("Report Materializer online.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			log.Println("Stopping Report Materializer service...")
			return
		case <-ticker.C:
			// Heartbeat
		}
	}
}
