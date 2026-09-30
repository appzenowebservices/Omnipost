<template>
  <form @submit.prevent="onSubmit">
    <div class="modal-card content" style="width: auto">
      <header class="modal-card-head">
        <p v-if="isEditing" class="has-text-grey-light is-size-7">
          {{ $t('globals.fields.id') }}: <copy-text :text="`${data.id}`" />
          {{ $t('globals.fields.uuid') }}: <copy-text :text="data.uuid" />
        </p>
        <b-tag v-if="isEditing" :class="[data.type, 'is-pulled-right']">
          {{ $t(`lists.types.${data.type}`) }}
        </b-tag>
        <h4 v-if="isEditing">
          {{ data.name }}
        </h4>
        <h4 v-else>
          {{ $t('lists.newList') }}
        </h4>
      </header>
      <section expanded class="modal-card-body">
        <b-field :label="$t('globals.fields.name')" label-position="on-border">
          <b-input :maxlength="200" :ref="'focus'" v-model="form.name" name="name"
            :placeholder="$t('globals.fields.name')" required />
        </b-field>

        <b-field :label="$t('lists.type')" label-position="on-border" :message="$t('lists.typeHelp')">
          <b-select v-model="form.type" name="type" :placeholder="$t('lists.typeHelp')" required expanded>
            <option value="private">
              {{ $t('lists.types.private') }}
            </option>
            <option value="public">
              {{ $t('lists.types.public') }}
            </option>
          </b-select>
        </b-field>

        <b-field :label="$t('lists.optin')" label-position="on-border" :message="$t('lists.optinHelp')">
          <b-select v-model="form.optin" name="optin" placeholder="Opt-in type" required expanded>
            <option value="single">
              {{ $t('lists.optins.single') }}
            </option>
            <option value="double">
              {{ $t('lists.optins.double') }}
            </option>
          </b-select>
        </b-field>

        <b-field :label="$t('globals.terms.tags')" label-position="on-border">
          <b-taginput v-model="form.tags" name="tags" ellipsis icon="tag-outline"
            :placeholder="$t('globals.terms.tags')" />
        </b-field>

        <b-field :label="$t('globals.fields.description')" label-position="on-border">
          <b-input :maxlength="2000" v-model="form.description" name="description" type="textarea"
            :placeholder="$t('globals.fields.description')" />
        </b-field>

        <b-field label="Confirm webhook URL" label-position="on-border"
          message="HTTPS endpoint notified when a subscriber confirms. Empty = disabled.">
          <b-input :maxlength="2000" v-model="form.webhookUrl" name="webhook_url" type="url"
            placeholder="https://example.com/hooks/omnipost-confirmed" />
          <p v-if="isEditing" class="help" style="margin-top: 4px;">
            Saved value:
            <copy-text v-if="data.webhookUrl" :text="data.webhookUrl" />
            <span v-else>—</span>
          </p>
        </b-field>

        <b-field label="Confirm webhook secret" label-position="on-border"
          message="HMAC signing secret. Leave blank to keep the saved one.">
          <b-input :maxlength="500" v-model="form.webhookSecret" name="webhook_secret" type="password"
            placeholder="••••••••" password-reveal />
          <p class="help" style="margin-top: 4px;">
            <a href="#" @click.prevent="onRevealSecret">Show saved secret</a>
            — loads it into this field where the eye icon toggles visibility.
          </p>
        </b-field>

        <b-field :message="$t('lists.archivedHelp')" :label="$t('lists.archived')">
          <b-switch v-model="isArchived" name="status" />
        </b-field>
      </section>
      <footer class="modal-card-foot has-text-right">
        <b-button @click="$parent.close()">
          {{ $t('globals.buttons.close') }}
        </b-button>
        <b-button v-if="isEditing && data.webhookUrl" @click="onTestWebhook" :loading="testingWebhook">
          Test webhook
        </b-button>
        <b-button v-if="$can('lists:manage_all') || $canList(data.id, 'list:manage')" native-type="submit"
          type="is-primary" :loading="loading.lists" data-cy="btn-save">
          {{ $t('globals.buttons.save') }}
        </b-button>
      </footer>
    </div>
  </form>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import CopyText from '../components/CopyText.vue';

export default Vue.extend({
  name: 'ListForm',

  components: {
    CopyText,
  },

  props: {
    data: { type: Object, default: () => ({}) },
    isEditing: { type: Boolean, default: false },
  },

  data() {
    return {
      testingWebhook: false,
      // Binds form input values.
      form: this.blankForm(),
    };
  },

  watch: {
    // Re-sync whenever a (different) row is passed in, so the form can
    // never show stale values from a previous edit.
    data: {
      immediate: true,
      handler(v) {
        this.resetForm(v || {});
      },
    },
  },

  methods: {
    blankForm() {
      return {
        name: '',
        type: 'private',
        optin: 'single',
        status: 'active',
        tags: [],
        webhookUrl: '',
        webhookSecret: '',
      };
    },

    resetForm(data) {
      this.form = { ...this.blankForm(), ...data };
      // Secrets are write-only: never carry a previously typed value over.
      this.form.webhookSecret = '';
    },

    onSubmit() {
      if (this.isEditing) {
        this.updateList();
        return;
      }

      this.createList();
    },

    createList() {
      const payload = {
        ...this.form,
        webhook_url: this.form.webhookUrl,
        webhook_secret: this.form.webhookSecret,
      };
      this.$api.createList(payload).then((data) => {
        this.$emit('finished');
        this.$parent.close();
        this.$utils.toast(this.$t('globals.messages.created', { name: data.name }));
      });
    },

    updateList() {
      const payload = {
        id: this.data.id,
        ...this.form,
        webhook_url: this.form.webhookUrl,
        webhook_secret: this.form.webhookSecret,
      };
      this.$api.updateList(payload).then((data) => {
        this.$emit('finished');
        this.$parent.close();
        this.$utils.toast(this.$t('globals.messages.updated', { name: data.name }));
      });
    },

    onTestWebhook() {
      this.testingWebhook = true;
      this.$api.testListWebhook(this.data.id).then(() => {
        this.testingWebhook = false;
        this.$utils.toast('Test event delivered');
      }).catch(() => {
        this.testingWebhook = false;
      });
    },

    onRevealSecret() {
      this.$api.getListWebhookSecret(this.data.id).then((data) => {
        this.form.webhookSecret = data.webhookSecret;
        this.$utils.toast('Saved secret loaded — eye icon toggles visibility');
      });
    },
  },

  computed: {
    ...mapState(['loading', 'profile']),

    isArchived: {
      get() {
        return this.form.status === 'archived';
      },
      set(v) {
        this.form.status = v ? 'archived' : 'active';
      },
    },
  },

  mounted() {
    this.$nextTick(() => {
      this.$refs.focus.focus();
    });
  },
});
</script>
