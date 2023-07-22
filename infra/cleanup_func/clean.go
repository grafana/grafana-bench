package cleanup_func

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/api/compute/v1"
)

// http handler for cloudfunction entrypoint
func RunClean(w http.ResponseWriter, r *http.Request) {
	err := cleanup()

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Successfully ran cleanup")
	}

	fmt.Fprintf(w, "Cleanup run")
}

// Does the cleanup
func cleanup() error {
	projectID := os.Getenv("PROJECT_ID")
	region := os.Getenv("REGION")
	zone := os.Getenv("ZONE")

	ctx := context.Background()
	computeService, err := compute.NewService(ctx)

	if err != nil {
		return fmt.Errorf("Error configuring compute service %w", err)
	}

	expiredInstances, err := getExpiredGrafanaInstances(context.Background(), computeService, projectID, zone)
	if err != nil {
		return err
	}

	if len(expiredInstances) == 0 {
		fmt.Println("No expired instances found")
	}

	for _, instance := range expiredInstances {
		err := removeBenchStateAssets(ctx, computeService, projectID, region, zone, instance.Name)
		if err != nil {
			fmt.Println("error deleting assets for instance:", instance.Name, err)
		} else {
			fmt.Println("successfully deleted assets for instance:", instance.Name)
		}
	}

	return nil
}

// Searches all instances across the project and returns an array of IDs for
// instances tagged for deletion
func getExpiredGrafanaInstances(ctx context.Context, computeService *compute.Service, projectID, zone string) ([]*compute.Instance, error) {
	expiredGrafanaInstances := []*compute.Instance{}

	var err error
	var token string
	var list *compute.InstanceAggregatedList

	// NOTE, not sure how pagetoken mechanism works. verify paginating correctly at some
	// point
	if list, err = computeService.Instances.AggregatedList(projectID).PageToken(token).Do(); err != nil {
		return expiredGrafanaInstances, err
	}

	currentDate := getDateFromTime(time.Now())
	fmt.Println("search for instances expired before:", currentDate)

	for _, instances := range list.Items {
		for _, instance := range instances.Instances {

			// filter out non grafana instances. We're using these as the
			// identifier
			if !strings.HasPrefix(instance.Name, "bench-grafana-instance") {
				continue
			}

			// check if expired
			expired, err := instanceExpired(currentDate, instance)
			if err != nil {
				return expiredGrafanaInstances, fmt.Errorf("Error checking if instance expired %w", err)
			}

			fmt.Println(instance.Name, "expired:", expired, "-", strings.Join(instance.Tags.Items, ","))
			if expired {
				expiredGrafanaInstances = append(expiredGrafanaInstances, instance)
			}
		}
	}

	return expiredGrafanaInstances, nil
}

// Removes firewall rule, grafana instance + static IP, and k6 instance + static IP
func removeBenchStateAssets(ctx context.Context, computeService *compute.Service, projectID, region, zone, instanceName string) error {
	// check if firewall rule exists
	_, err := computeService.Firewalls.Delete(projectID, instanceName).Do()
	if err != nil {
		return fmt.Errorf("failed to delete firewall rule: %v", err)
	}
	fmt.Println("firewall rule deleted:", instanceName)

	// delete instance
	_, err = computeService.Instances.Delete(projectID, zone, instanceName).Do()
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}
	fmt.Println("grafana instance deleted:", instanceName)

	// delete static IP
	_, err = computeService.Addresses.Delete(projectID, region, instanceName).Do()
	if err != nil {
		return fmt.Errorf("failed to delete static IP: %v", err)
	}
	fmt.Println("grafana static ip deleted:", instanceName)

	// k6
	// bench-grafana-instance-{identifer} -> bench-k6-instance-{identifier}
	k6InstanceName := strings.Replace(instanceName, "grafana", "k6", -1)

	// delete k6 instance
	_, err = computeService.Instances.Delete(projectID, zone, k6InstanceName).Do()
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}
	fmt.Println("k6 instance deleted:", k6InstanceName)

	// delete k6 static ip
	_, err = computeService.Addresses.Delete(projectID, region, k6InstanceName).Do()
	if err != nil {
		return fmt.Errorf("failed to delete static IP: %v", err)
	}
	fmt.Println("k6 static ip deleted:", k6InstanceName)

	return nil
}

func getDateFromTime(date time.Time) time.Time {
	year, month, day := date.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, date.Location())
}

// Loops through all the tags for *compute.Instance searching for
// expire-data-YYYY-MM-DD and checks if the data is before the current_date.
func instanceExpired(currentDate time.Time, instance *compute.Instance) (bool, error) {
	// get the expire tag
	for _, tag := range instance.Tags.Items {
		// expire-date-2021-03-01
		if strings.HasPrefix(tag, "expire-date-") {
			dateString := strings.Split(tag, "expire-date-")[1]

			expirationDate, err := time.Parse("2006-01-02", dateString)
			if err != nil {
				fmt.Println("Error parsing date from tag", tag, err)
				continue
			}

			expirationDate = getDateFromTime(expirationDate)
			if currentDate.After(expirationDate) {
				return true, nil
			}
		}
	}
	return false, nil
}
