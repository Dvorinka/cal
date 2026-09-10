package dev.cal.app;

import android.appwidget.AppWidgetManager;
import android.appwidget.AppWidgetProvider;
import android.content.Context;
import android.content.SharedPreferences;
import android.widget.RemoteViews;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.Locale;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Home-screen widget: today's agenda, fetched from /api/widget/today with the
 * read-only widget token the app writes to SharedPreferences via WidgetConfig.
 */
public class CalWidgetProvider extends AppWidgetProvider {

    private static final ExecutorService EXEC = Executors.newSingleThreadExecutor();

    @Override
    public void onUpdate(Context context, AppWidgetManager mgr, int[] ids) {
        SharedPreferences prefs = context.getSharedPreferences("cal", Context.MODE_PRIVATE);
        String server = prefs.getString("server", "");
        String token = prefs.getString("widgetToken", "");
        SimpleDateFormat fmt = new SimpleDateFormat("EEE, MMM d", Locale.getDefault());
        String date = fmt.format(new Date());

        for (int id : ids) {
            RemoteViews views = new RemoteViews(context.getPackageName(), R.layout.cal_widget);
            views.setTextViewText(R.id.widget_date, date);
            views.setTextViewText(R.id.widget_lines, "Loading…");
            mgr.updateAppWidget(id, views);

            if (server.isEmpty() || token.isEmpty()) {
                views.setTextViewText(R.id.widget_lines, "Open Cal once to connect the widget.");
                mgr.updateAppWidget(id, views);
                continue;
            }
            EXEC.execute(() -> fetch(context, mgr, id, server, token));
        }
    }

    private void fetch(Context context, AppWidgetManager mgr, int id, String server, String token) {
        RemoteViews views = new RemoteViews(context.getPackageName(), R.layout.cal_widget);
        try {
            URL url = new URL(server + "/api/widget/today?token=" + token);
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setConnectTimeout(8000);
            conn.setReadTimeout(8000);
            if (conn.getResponseCode() != 200) {
                conn.disconnect();
                views.setTextViewText(R.id.widget_lines, "Could not load (check the widget token).");
                mgr.updateAppWidget(id, views);
                return;
            }
            BufferedReader in = new BufferedReader(new InputStreamReader(conn.getInputStream()));
            StringBuilder body = new StringBuilder();
            String line;
            while ((line = in.readLine()) != null) body.append(line);
            in.close();
            conn.disconnect();

            JSONArray entries = new JSONArray(body.toString());
            StringBuilder lines = new StringBuilder();
            for (int i = 0; i < entries.length() && i < 6; i++) {
                JSONObject e = entries.getJSONObject(i);
                String time = e.optString("startTime", "");
                boolean done = e.optBoolean("completed", false);
                if (!time.isEmpty()) lines.append(time).append("  ");
                lines.append(done ? "✓ " : "• ").append(e.optString("title", "")).append("\n");
            }
            views.setTextViewText(
                R.id.widget_lines,
                lines.length() == 0 ? "Nothing today — a quiet page." : lines.toString().trim()
            );
        } catch (Exception e) {
            views.setTextViewText(R.id.widget_lines, "Offline — tap to open Cal.");
        }
        mgr.updateAppWidget(id, views);
    }
}
