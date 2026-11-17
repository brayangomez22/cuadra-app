import type { PostHogConfig } from 'posthog-js';
import { cleanUrl } from '../model/cleanUrl';

// The URL properties PostHog adds to every event; their query may carry what
// the user searched.
const urlProps = ['$current_url', '$referrer', '$initial_referrer', '$pathname'];

/**
 * PostHog with nothing collected on its own: only the events of the catalog
 * (docs/eventos.md). No autocapture (it reads the text of what is clicked),
 * no session recording, heatmaps, surveys or remote code, no cookies, and
 * URLs without their query.
 */
export function posthogOptions(host = 'https://us.i.posthog.com'): Partial<PostHogConfig> {
  return {
    api_host: host,
    autocapture: false,
    capture_pageview: false,
    capture_pageleave: false,
    capture_dead_clicks: false,
    capture_exceptions: false,
    capture_performance: false,
    capture_heatmaps: false,
    rageclick: false,
    disable_session_recording: true,
    disable_surveys: true,
    disable_web_experiments: true,
    disable_external_dependency_loading: true,
    advanced_disable_flags: true,
    mask_all_text: true,
    persistence: 'memory',
    save_referrer: false,
    person_profiles: 'identified_only',
    before_send: (event) => {
      for (const prop of urlProps) {
        const value: unknown = event.properties[prop];
        if (typeof value === 'string') event.properties[prop] = cleanUrl(value);
      }
      return event;
    },
  };
}
