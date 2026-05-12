// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri_plugin_shell::ShellExt;

fn main() {
  tauri::Builder::default()
    .plugin(tauri_plugin_shell::init())
    .setup(|app| {
      // Spawn the Go backend sidecar
      // The name matches the entry in tauri.conf.json "externalBin"
      let shell = app.shell();
      let sidecar = shell.sidecar("drivehive-backend")
        .map_err(|e| e.to_string())?
        .args(["-port", "8080"]);

      let (_rx, _child) = sidecar.spawn()
        .map_err(|e| e.to_string())?;

      Ok(())
    })
    .run(tauri::generate_context!())
    .expect("error while running tauri application");
}
