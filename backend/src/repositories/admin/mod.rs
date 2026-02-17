//! Admin domain repositories

pub mod audit_repo;
pub mod rbac_repo;

// Backward compatibility: re-export modules with original names
pub use audit_repo as admin_audit_repo;
pub use rbac_repo as admin_rbac_repo;
