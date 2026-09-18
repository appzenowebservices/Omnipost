<template>
  <section class="notifications content">
    <h1 class="title is-4">
      {{ $t('notifications.title') }}
    </h1>
    <p class="is-size-7 has-text-grey">
      Browser web-push via Firebase Cloud Messaging. Enable this browser below,
      then send a test. Production must be served over HTTPS
      (plain <code>http://localhost</code> is fine for testing).
    </p>
    <hr />

    <div class="box">
      <h2 class="title is-6">This browser</h2>
      <b-field grouped group-multiline>
        <div class="control">
          <b-tag :type="status.supported ? 'is-success' : 'is-danger'">
            {{ status.supported ? 'Push supported' : 'Push unsupported' }}
          </b-tag>
        </div>
        <div class="control">
          <b-tag :type="status.secure ? 'is-success' : 'is-danger'">
            {{ status.secure ? 'Secure context' : 'Insecure context' }}
          </b-tag>
        </div>
        <div class="control">
          <b-tag type="is-info">Permission: {{ status.permission }}</b-tag>
        </div>
        <div class="control">
          <b-tag :type="status.configured ? 'is-success' : 'is-warning'">
            {{ status.configured ? 'Server configured' : 'Server not configured' }}
          </b-tag>
        </div>
      </b-field>

      <b-message v-if="!status.configured" type="is-warning" size="is-small">
        The server has no Firebase web config. Set <code>FIREBASE_WEB_*</code> and
        <code>FIREBASE_VAPID_KEY</code> from the Firebase Console (see <code>.env.sample</code>)
        and restart the backend.
      </b-message>

      <div class="field is-grouped">
        <div class="control">
          <b-button type="is-primary" :loading="busy" :disabled="!status.supported"
            @click="enable">
            Enable notifications on this browser
          </b-button>
        </div>
      </div>

      <b-field v-if="token" label="FCM token (this browser)" label-position="on-border">
        <b-input :value="shortToken" readonly />
      </b-field>

      <b-message v-if="foregroundNote" type="is-info" size="is-small" has-icon>
        {{ foregroundNote }}
      </b-message>
    </div>

    <div class="box">
      <h2 class="title is-6">Send a test</h2>
      <form @submit.prevent="sendTest">
        <b-field label="Title" label-position="on-border">
          <b-input v-model="form.title" placeholder="OmniPost test" maxlength="200" />
        </b-field>
        <b-field label="Body" label-position="on-border">
          <b-input v-model="form.body" type="textarea" placeholder="Hello from OmniPost" maxlength="2000" />
        </b-field>
        <b-field>
          <b-checkbox v-model="form.onlyThisBrowser">
            Send only to this browser
          </b-checkbox>
        </b-field>
        <b-button native-type="submit" type="is-primary" :loading="busy">
          Send test notification
        </b-button>
      </form>
      <b-message v-if="sendResult" type="is-success" size="is-small" class="mt-4">
        Delivered: {{ sendResult.success_count }}, failed: {{ sendResult.failure_count }},
        deactivated stale tokens: {{ sendResult.deactivated }}.
        Keep this tab open to catch the foreground message, or close it to test the background path.
      </b-message>
    </div>

    <div class="box">
      <h2 class="title is-6">Registered browsers</h2>
      <b-table :data="tokens" :loading="busy" default-sort="created_at" default-sort-direction="desc">
        <b-table-column field="label" label="Label" v-slot="props">
          {{ props.row.label || '—' }}
        </b-table-column>
        <b-table-column field="token" label="Token" v-slot="props">
          <code>{{ truncate(props.row.token) }}</code>
        </b-table-column>
        <b-table-column field="is_active" label="Active" v-slot="props">
          <b-tag :type="props.row.is_active ? 'is-success' : 'is-danger'" size="is-small">
            {{ props.row.is_active ? 'yes' : 'no' }}
          </b-tag>
        </b-table-column>
        <b-table-column field="created_at" label="Registered" sortable v-slot="props">
          {{ props.row.created_at }}
        </b-table-column>
        <b-table-column label="" v-slot="props" width="60">
          <a href="#" @click.prevent="removeToken(props.row.id)">
            <b-icon icon="trash-can-outline" />
          </a>
        </b-table-column>
        <template #empty>
          <div class="has-text-centered">No browsers registered yet.</div>
        </template>
      </b-table>
      <p class="is-size-7 has-text-grey mt-2">
        Tokens FCM reports as unregistered/invalid are automatically marked inactive on send.
      </p>
    </div>
  </section>
</template>

<script>
import Vue from 'vue';
import {
  enablePush, getFirebaseConfig, getPermissionState, isPushSupported,
  isSecureContextOk, onForegroundPush,
} from '../lib/firebase-push';

export default Vue.extend({
  data() {
    return {
      busy: false,
      token: '',
      tokens: [],
      sendResult: null,
      foregroundNote: '',
      form: {
        title: 'OmniPost test',
        body: 'Hello from OmniPost',
        onlyThisBrowser: true,
      },
      status: {
        supported: false,
        secure: false,
        permission: 'unknown',
        configured: false,
      },
      unsubscribe: null,
    };
  },

  methods: {
    refreshStatus() {
      this.status.supported = isPushSupported();
      this.status.secure = isPushSupported() ? isSecureContextOk() : false;
      this.status.permission = getPermissionState();
    },

    async refreshConfig() {
      try {
        const cfg = await getFirebaseConfig();
        this.status.configured = Boolean(cfg && cfg.vapid_key && cfg.project_id);
      } catch (e) {
        this.status.configured = false;
      }
    },

    async enable() {
      this.busy = true;
      try {
        const token = await enablePush();
        this.token = token;
        await this.$api.savePushToken({ token, label: navigator.userAgent.slice(0, 120) });
        this.$utils.toast('Push enabled — token saved');
        this.refreshStatus();
        this.refreshTokens();
        this.listenForeground();
      } catch (e) {
        this.$utils.toast(e.message || e.toString(), 'is-danger');
      } finally {
        this.busy = false;
      }
    },

    async listenForeground() {
      if (this.unsubscribe || !this.token) {
        return;
      }
      try {
        this.unsubscribe = await onForegroundPush((payload) => {
          const n = payload.notification || {};
          this.foregroundNote = `Foreground message received: ${n.title || ''} ${n.body || ''}`.trim();
          this.$utils.toast(`Push: ${n.title || 'new notification'}`);
        });
      } catch (e) {
        // Foreground listener is best-effort; background path still works.
      }
    },

    async sendTest() {
      if (!this.form.body.trim()) {
        this.$utils.toast('Body is required', 'is-danger');
        return;
      }
      this.busy = true;
      this.sendResult = null;
      try {
        const data = await this.$api.sendTestPush({
          title: this.form.title,
          body: this.form.body,
          token: this.form.onlyThisBrowser ? this.token : '',
        });
        this.sendResult = data;
        this.$utils.toast('Test push sent');
        this.refreshTokens();
      } catch (e) {
        // Error toast is shown by the API interceptor; keep form state.
      } finally {
        this.busy = false;
      }
    },

    async refreshTokens() {
      try {
        this.tokens = await this.$api.getPushTokens();
      } catch (e) {
        // Non-settings roles cannot list tokens; the page still works
        // for enabling this browser.
        this.tokens = [];
      }
    },

    async removeToken(id) {
      this.$utils.confirm(null, () => {
        this.$api.deletePushToken(id).then(() => this.refreshTokens());
      });
    },

    truncate(t) {
      if (!t) {
        return '';
      }
      return t.length > 24 ? `${t.slice(0, 12)}…${t.slice(-8)}` : t;
    },
  },

  computed: {
    shortToken() {
      return this.truncate(this.token);
    },
  },

  mounted() {
    this.refreshStatus();
    this.refreshConfig();
    this.refreshTokens();
  },

  destroyed() {
    if (this.unsubscribe) {
      this.unsubscribe();
    }
  },
});
</script>
