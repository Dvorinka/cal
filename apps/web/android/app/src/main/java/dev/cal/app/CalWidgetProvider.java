package dev.cal.app;

import android.app.PendingIntent;
import android.appwidget.AppWidgetManager;
import android.appwidget.AppWidgetProvider;
import android.content.ComponentName;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.graphics.Paint;
import android.view.View;
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
 * Tap opens the app; the refresh icon refetches; done items are struck through.
 */
public class CalWidgetProvider extends AppWidgetProvider {

    private static final String ACTION_REFRESH = "dev.cal.app.WIDGET_REFRESH";
    private static final int MAX_ROWS = 6;
    private static final ExecutorService EXEC = Executors.newSingleThreadExecutor();

    /** Public so WidgetConfigPlugin can poke widgets after login. */
    public static void requestRefresh(Context context) {
        Intent i = new Intent(context, CalWidgetProvider.class).setAction(ACTION_REFRESH);
        context.sendBroadcast(i);
    }

    @Override
    public void onReceive(Context context, Intent intent) {
        super.onReceive(context, intent);
        if (ACTION_REFRESH.equals(intent.getAction())) {
            AppWidgetManager mgr = AppWidgetManager.getInstance(context);
            int[] ids = mgr.getAppWidgetIds(
                new ComponentName(context, CalWidgetProvider.class));
            if (ids.length > 0) onUpdate(context, mgr, ids);
        }
    }

    @Override
    public void onUpdate(Context context, AppWidgetManager mgr, int[] ids) {
        SharedPreferences prefs = context.getSharedPreferences("cal", Context.MODE_PRIVATE);
        String server = prefs.getString("server", "");
        String token = prefs.getString("widgetToken", "");
        String date = new SimpleDateFormat("EEE, MMM d", Locale.getDefault()).format(new Date());

        for (int id : ids) {
            RemoteViews views = shell(context);
            views.setTextViewText(R.id.widget_date, date);
            setStatus(views, context.getString(R.string.widget_loading));
            mgr.updateAppWidget(id, views);

            if (server.isEmpty() || token.isEmpty()) {
                RemoteViews empty = shell(context);
                empty.setTextViewText(R.id.widget_date, date);
                setStatus(empty, context.getString(R.string.widget_connect_hint));
                mgr.updateAppWidget(id, empty);
                continue;
            }
            EXEC.execute(() -> fetch(context, mgr, id, server, token, date));
        }
    }

    private RemoteViews shell(Context context) {
        RemoteViews views = new RemoteViews(context.getPackageName(), R.layout.cal_widget);

        Intent open = context.getPackageManager().getLaunchIntentForPackage(context.getPackageName());
        if (open != null) {
            PendingIntent pi = PendingIntent.getActivity(context, 0, open,
                PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
            views.setOnClickPendingIntent(R.id.widget_root, pi);
        }
        Intent refresh = new Intent(context, CalWidgetProvider.class).setAction(ACTION_REFRESH);
        views.setOnClickPendingIntent(R.id.widget_refresh, PendingIntent.getBroadcast(
            context, 1, refresh, PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE));
        return views;
    }

    private void setStatus(RemoteViews views, String text) {
        views.setViewVisibility(R.id.widget_status, View.VISIBLE);
        views.setTextViewText(R.id.widget_status, text);
    }

    private void fetch(Context context, AppWidgetManager mgr, int id,
                       String server, String token, String date) {
        RemoteViews views = shell(context);
        views.setTextViewText(R.id.widget_date, date);
        try {
            URL url = new URL(server + "/api/widget/today?token=" + token);
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setConnectTimeout(8000);
            conn.setReadTimeout(8000);
            if (conn.getResponseCode() != 200) {
                conn.disconnect();
                setStatus(views, context.getString(R.string.widget_bad_token));
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
            views.removeAllViews(R.id.widget_list);
            views.setViewVisibility(R.id.widget_status, View.GONE);
            if (entries.length() == 0) {
                setStatus(views, context.getString(R.string.widget_empty));
            }
            for (int i = 0; i < entries.length() && i < MAX_ROWS; i++) {
                JSONObject e = entries.getJSONObject(i);
                RemoteViews row = new RemoteViews(context.getPackageName(), R.layout.cal_widget_row);
                row.setTextViewText(R.id.row_time, e.optString("startTime", ""));
                row.setTextViewText(R.id.row_title, e.optString("title", ""));
                if (e.optBoolean("completed", false)) {
                    row.setInt(R.id.row_title, "setPaintFlags",
                        Paint.STRIKE_THRU_TEXT_FLAG | Paint.ANTI_ALIAS_FLAG);
                    row.setTextColor(R.id.row_title,
                        context.getColor(R.color.widget_done));
                }
                views.addView(R.id.widget_list, row);
            }
            if (entries.length() > MAX_ROWS) {
                setStatus(views, "+" + (entries.length() - MAX_ROWS) + " more — tap to open");
            }
        } catch (Exception e) {
            setStatus(views, context.getString(R.string.widget_offline));
        }
        mgr.updateAppWidget(id, views);
    }
}
