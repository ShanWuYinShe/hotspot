use axum::{routing::post, Router};

use crate::{handlers::report::ReportHandler, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/create", post(ReportHandler::create_report))
        .route("/list", post(ReportHandler::get_report_history))
        .route("/list_all", post(ReportHandler::get_all_reports))
        .route("/handle", post(ReportHandler::handle_report))
}

