use axum::{routing::post, Router};

use crate::{handlers::user, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/discover", post(user::discover))
}




