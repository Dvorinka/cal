package dev.cal.app;

import android.content.Context;
import android.content.SharedPreferences;

import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

/**
 * Writes the server origin + read-only widget token into SharedPreferences so
 * CalWidgetProvider can fetch /api/widget/today without a session cookie.
 */
@CapacitorPlugin(name = "WidgetConfig")
public class WidgetConfigPlugin extends Plugin {

    @PluginMethod
    public void set(PluginCall call) {
        String server = call.getString("server", "");
        String token = call.getString("widgetToken", "");
        SharedPreferences prefs = getContext().getSharedPreferences("cal", Context.MODE_PRIVATE);
        prefs.edit().putString("server", server).putString("widgetToken", token).apply();
        call.resolve();
    }
}
