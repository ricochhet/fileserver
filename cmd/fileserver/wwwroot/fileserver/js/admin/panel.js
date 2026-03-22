/** Returns the full HTML string for the admin drawer panel. */
export function buildPanelHTML() {
	return `
<div id="slv-admin-panel" class="slv-admin-panel slv-hidden" role="dialog" aria-modal="true">
  <div id="slv-admin-backdrop" class="slv-overlay-backdrop"></div>
  <div class="slv-admin-drawer">

    <div class="slv-admin-drawer-header">
      <div class="slv-admin-drawer-title">
        <span class="slv-admin-badge" data-i18n="admin_badge">admin</span>
      </div>
      <button id="slv-admin-close" class="slv-btn secondary">✕</button>
    </div>

    <div class="slv-admin-tabs" role="tablist">
      <button class="slv-admin-tab active" data-tab="users" role="tab" aria-selected="true"
        data-i18n="admin_tab_users">users</button>
      <button class="slv-admin-tab" data-tab="channels" role="tab" aria-selected="false"
        data-i18n="admin_tab_channels">channels</button>
      <button class="slv-admin-tab" data-tab="upload" role="tab" aria-selected="false"
        data-i18n="admin_tab_upload">upload</button>
    </div>

    <div class="slv-admin-body">

      <!-- ── Users pane ─────────────────────────────────────── -->
      <div class="slv-admin-pane" id="slv-admin-pane-users">

        <section class="slv-admin-section">
          <h3 data-i18n="admin_create_user_heading">create user</h3>

          <div class="slv-login-field">
            <label for="slv-au-username" data-i18n="login_username_label">username</label>
            <input type="text" id="slv-au-username" autocomplete="off" spellcheck="false" />
          </div>

          <div class="slv-login-field">
            <label for="slv-au-password" data-i18n="login_password_label">password</label>
            <input type="text" id="slv-au-password" autocomplete="off" spellcheck="false" />
          </div>

          <div class="slv-login-field">
            <label for="slv-au-display" data-i18n="admin_display_name_label">
              display name
              <span class="slv-chat-optional" data-i18n="chat_optional">(optional)</span>
            </label>
            <input type="text" id="slv-au-display" autocomplete="off" />
          </div>

          <div class="slv-admin-checkbox-row">
            <input type="checkbox" id="slv-au-admin" />
            <label for="slv-au-admin" data-i18n="admin_is_admin_label">admin</label>
          </div>

          <div id="slv-au-error" class="slv-login-error slv-hidden"></div>
          <div id="slv-au-ok"    class="slv-admin-ok    slv-hidden"></div>

          <div class="slv-admin-form-actions">
            <button id="slv-au-submit" class="slv-button" data-i18n="admin_create_user_submit">
              create user →
            </button>
          </div>
        </section>

        <section class="slv-admin-section">
          <div class="slv-admin-section-head">
            <h3 data-i18n="admin_users_heading">all users</h3>
            <button id="slv-au-refresh" class="slv-btn secondary" data-i18n="admin_refresh">↺</button>
          </div>
          <div id="slv-admin-user-list" class="slv-admin-user-list">
            <span class="slv-admin-loading" data-i18n="admin_loading">loading…</span>
          </div>
        </section>

      </div>

      <!-- ── Channels pane ──────────────────────────────────── -->
      <div class="slv-admin-pane slv-hidden" id="slv-admin-pane-channels">

        <section class="slv-admin-section">
          <h3 data-i18n="admin_create_channel_heading">create channel</h3>

          <div class="slv-login-field">
            <label for="slv-ac-code" data-i18n="admin_channel_code_label">channel code</label>
            <input type="text" id="slv-ac-code" autocomplete="off" spellcheck="false"
              data-i18n-placeholder="admin_channel_code_placeholder" placeholder="e.g. project-alpha" />
          </div>

          <div class="slv-login-field">
            <label for="slv-ac-name" data-i18n="admin_channel_name_label">
              channel name
              <span class="slv-chat-optional" data-i18n="chat_optional">(optional)</span>
            </label>
            <input type="text" id="slv-ac-name" autocomplete="off"
              data-i18n-placeholder="admin_channel_name_placeholder" placeholder="e.g. Project Alpha" />
          </div>

          <div id="slv-ac-error" class="slv-login-error slv-hidden"></div>
          <div id="slv-ac-ok"    class="slv-admin-ok    slv-hidden"></div>

          <div class="slv-admin-form-actions">
            <button id="slv-ac-submit" class="slv-button" data-i18n="admin_create_channel_submit">
              create channel →
            </button>
          </div>
        </section>

        <section class="slv-admin-section">
          <div class="slv-admin-section-head">
            <h3 data-i18n="admin_channels_heading">all channels</h3>
            <button id="slv-ac-refresh" class="slv-btn secondary" data-i18n="admin_refresh">↺</button>
          </div>
          <div id="slv-admin-channel-list" class="slv-admin-user-list">
            <span class="slv-admin-loading" data-i18n="admin_loading">loading…</span>
          </div>
        </section>

      </div>

      <!-- ── Upload pane ────────────────────────────────────── -->
      <div class="slv-admin-pane slv-hidden" id="slv-admin-pane-upload">

        <section class="slv-admin-section">
          <h3 data-i18n="admin_upload_heading">upload files</h3>

          <div class="slv-login-field">
            <label for="slv-up-path" data-i18n="admin_upload_path_label">
              destination path
              <span class="slv-chat-optional" data-i18n="chat_optional">(optional)</span>
            </label>
            <input type="text" id="slv-up-path" autocomplete="off" spellcheck="false"
              data-i18n-placeholder="admin_upload_path_placeholder" placeholder="e.g. reports/2025" />
          </div>

          <div class="slv-login-field">
            <label data-i18n="admin_upload_files_label">files</label>
            <div id="slv-up-drop" class="slv-admin-dropzone">
              <label for="slv-up-input" class="slv-admin-dropzone-label">
                <input type="file" id="slv-up-input" class="slv-admin-file-input" multiple />
                <span class="slv-admin-dropzone-icon">⬆</span>
                <span data-i18n="admin_upload_drop_hint">drop files here or click to browse</span>
              </label>
              <div id="slv-up-filelist" class="slv-admin-filelist"></div>
            </div>
          </div>

          <div id="slv-up-error" class="slv-login-error slv-hidden"></div>
          <div id="slv-up-ok"    class="slv-admin-ok    slv-hidden"></div>

          <div id="slv-up-progress" class="slv-admin-progress slv-hidden">
            <div class="slv-admin-progress-bar" id="slv-up-bar"></div>
          </div>

          <div class="slv-admin-form-actions">
            <button id="slv-up-submit" class="slv-button" data-i18n="admin_upload_submit">
              upload →
            </button>
            <button id="slv-up-clear" class="slv-button slv-danger" data-i18n="admin_upload_clear">
              clear
            </button>
          </div>
        </section>

      </div>
    </div>
  </div>
</div>`;
}
