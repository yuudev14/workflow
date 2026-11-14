package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tickNow has the kind of jobs.ScheduleTickArgs but no unique rule, so the test does not
// wait for the clock of the worker
type tickNow struct{}

func (tickNow) Kind() string { return "schedule_tick" }

func TestSchedules(t *testing.T) {
	const base = "/api/v1/schedules"
	c := admin(t)
	playbookID := newPlaybook(t, c)
	c.expect(t, http.StatusOK, http.MethodPut, "/api/v1/playbooks/tasks/"+playbookID, J{
		"nodes": []J{{"name": "start", "operation": "start", "parameters": J{}}},
		"edges": J{},
	})
	schedule := J{"name": unique("nightly"), "playbook_id": playbookID, "cron": "0 2 * * *", "timezone": "Europe/Paris"}
	id := c.create(t, base, schedule)["id"].(string)

	t.Run("POST /schedules", func(t *testing.T) {
		webhook := c.expect(t, http.StatusOK, http.MethodPost, "/api/v1/playbooks", J{"name": unique("hook"), "trigger_type": "webhook"}).(J)["id"]
		c.expectAll(t, []request{
			{"returns 400 when the cron is a short form", http.MethodPost, base, J{"name": "x", "playbook_id": playbookID, "cron": "@daily"}, http.StatusBadRequest},
			{"returns 400 when the cron has 6 fields", http.MethodPost, base, J{"name": "x", "playbook_id": playbookID, "cron": "0 0 2 * * *"}, http.StatusBadRequest},
			{"returns 400 when the time zone is unknown", http.MethodPost, base, J{"name": "x", "playbook_id": playbookID, "cron": "0 2 * * *", "timezone": "Mars/Base"}, http.StatusBadRequest},
			{"returns 400 when the playbook is not manual", http.MethodPost, base, J{"name": "x", "playbook_id": webhook, "cron": "0 2 * * *"}, http.StatusBadRequest},
			{"returns 404 when the playbook does not exist", http.MethodPost, base, J{"name": "x", "playbook_id": uuid.NewString(), "cron": "0 2 * * *"}, http.StatusNotFound},
		})

		t.Run("returns 403 without the grant to run playbooks", func(t *testing.T) {
			roleID := c.create(t, "/api/v1/roles", J{"name": unique("role"), "permissions": J{"schedules": []string{"read", "create"}}})["id"].(string)
			_, user := newUser(t, roleID)
			user.expect(t, http.StatusForbidden, http.MethodPost, base, schedule)
		})

		t.Run("saves the next run time in the time zone", func(t *testing.T) {
			got := c.get(t, base+"/"+id).(J)
			assert.Equal(t, "Europe/Paris", got["timezone"])
			next, err := time.Parse(time.RFC3339, got["next_run_at"].(string))
			require.NoError(t, err)
			assert.True(t, next.After(time.Now()))
			assert.Contains(t, []int{0, 1}, next.UTC().Hour()) // 02:00 in Paris is 00:00 or 01:00 UTC
		})
	})

	t.Run("GET /schedules", func(t *testing.T) {
		other := newPlaybook(t, c)
		c.create(t, base, J{"name": unique("a"), "playbook_id": other, "cron": "0 3 * * *"})
		c.create(t, base, J{"name": unique("b"), "playbook_id": other, "cron": "0 4 * * *"})

		t.Run("returns a page of entries and the total", func(t *testing.T) {
			page := c.get(t, base+"?playbook_id="+other+"&limit=1&offset=1").(J)
			assert.Len(t, items(page), 1)
			assert.EqualValues(t, 2, page["total"])
		})
		c.expectAll(t, []request{
			{"returns 400 when the limit is above 200", http.MethodGet, base + "?limit=201", nil, http.StatusBadRequest},
			{"returns 400 when the playbook id is not a uuid", http.MethodGet, base + "?playbook_id=x", nil, http.StatusBadRequest},
		})
	})

	t.Run("POST /schedules/preview", func(t *testing.T) {
		t.Run("returns the next 3 times", func(t *testing.T) {
			got := c.expect(t, http.StatusOK, http.MethodPost, base+"/preview", J{"cron": "*/5 * * * *"}).(J)
			assert.Len(t, items(got["next_runs"]), 3)
		})
		c.expectAll(t, []request{
			{"returns 400 when the cron is bad", http.MethodPost, base + "/preview", J{"cron": "x"}, http.StatusBadRequest},
		})
	})

	t.Run("PATCH /schedules/:id", func(t *testing.T) {
		c.expectAll(t, []request{
			{"returns 400 when the new cron is bad", http.MethodPatch, base + "/" + id, J{"cron": "@hourly"}, http.StatusBadRequest},
			{"returns 404 when the schedule does not exist", http.MethodPatch, base + "/" + uuid.NewString(), J{"name": "x"}, http.StatusNotFound},
		})
		t.Run("disables the schedule", func(t *testing.T) {
			got := c.expect(t, http.StatusOK, http.MethodPatch, base+"/"+id, J{"enabled": false}).(J)
			assert.Equal(t, false, got["enabled"])
		})
	})

	t.Run("the tick", func(t *testing.T) {
		tickID := c.create(t, base, J{"name": unique("tick"), "playbook_id": playbookID, "cron": "0 0 1 1 *", "parameters": J{"env": "prod"}})["id"].(string)
		due := time.Now().Add(-time.Minute).UTC().Truncate(time.Second)
		_, err := testPool.Exec(context.Background(), `UPDATE schedulers SET next_run_at = $2 WHERE id = $1`, tickID, due)
		require.NoError(t, err)

		client, err := river.NewClient(riverpgxv5.New(testPool), &river.Config{})
		require.NoError(t, err)
		_, err = client.Insert(context.Background(), tickNow{}, nil)
		require.NoError(t, err)

		var run J
		require.Eventually(t, func() bool {
			for _, r := range items(c.get(t, "/api/v1/playbooks/history?playbook_id="+playbookID)) {
				if r.(J)["schedule_id"] == tickID {
					run = r.(J)
					return true
				}
			}
			return false
		}, 30*time.Second, 200*time.Millisecond, "the tick did not queue a run")

		assert.Equal(t, "manual", run["trigger_type"])
		assert.Nil(t, run["triggered_by"])
		params := run["input"].(J)["parameters"].(J)
		assert.Equal(t, "prod", params["env"])
		assert.Equal(t, due.Format(time.RFC3339), params["scheduled_at"])

		got := c.get(t, base+"/"+tickID).(J)
		assert.Equal(t, got["name"], run["schedule_name"])
		assert.Equal(t, due.Format(time.RFC3339), got["last_run_at"])
		next, err := time.Parse(time.RFC3339, got["next_run_at"].(string))
		require.NoError(t, err)
		assert.True(t, next.After(time.Now()))
	})

	t.Run("DELETE /schedules/:id", func(t *testing.T) {
		c.expectAll(t, []request{
			{"deletes the schedule", http.MethodDelete, base + "/" + id, nil, http.StatusOK},
			{"returns 404 when the schedule is gone", http.MethodGet, base + "/" + id, nil, http.StatusNotFound},
			{"returns 404 when the schedule is deleted again", http.MethodDelete, base + "/" + id, nil, http.StatusNotFound},
		})
	})
}
