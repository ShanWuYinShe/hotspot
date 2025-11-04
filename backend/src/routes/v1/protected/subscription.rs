use axum::{routing::post, Router};

use crate::{handlers::subscription::SubscriptionHandler, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/get", post(SubscriptionHandler::get_subscriptions))
        .route("/update", post(SubscriptionHandler::update_subscription))
        .route("/update_batch", post(SubscriptionHandler::update_subscriptions_batch))
}

