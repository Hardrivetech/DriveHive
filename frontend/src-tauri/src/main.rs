// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri::Manager;
use tauri_plugin_shell::{ShellExt, process::CommandEvent};

fn main() {
  tauri::Builder::default()
    .plugin(tauri_plugin_shell::init())
    .setup(|app| {
      // Spawn the Go backend sidecar
      // The name matches the entry in tauri.conf.json "externalBin"
      let shell = app.shell();
      
      // Get a persistent data directory for the DB
      let data_dir = app.path().app_data_dir().unwrap_or_else(|_| std::path::PathBuf::from("./"));
      std::fs::create_dir_all(&data_dir).ok();
      let db_path = data_dir.join("drivehive.db");

      let sidecar = shell.sidecar("drivehive-backend")
        .map_err(|e| e.to_string())?
        .args(["-port", "8080", "-db", db_path.to_str().unwrap()]);

      let (mut rx, child) = sidecar.spawn()
        .map_err(|e| e.to_string())?;

      tauri::async_runtime::spawn(async move {
        // Keep the child handle alive so the sidecar process doesn't exit immediately
        let _child_handle = child;
        while let Some(event) = rx.recv().await {
          if let CommandEvent::Stdout(line) = event {
            // Forward Go logs to the terminal
            println!("Backend: {}", String::from_utf8_lossy(&line));
          }
        }
      });

      Ok(())
    })
    .run(tauri::generate_context!())
    .expect("error while running tauri application");
}
