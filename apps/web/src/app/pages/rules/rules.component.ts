import { Component, OnInit, inject, signal } from '@angular/core';
import { RuleService } from '../../core/services/rule.service';
import { DeviceService } from '../../core/services/device.service';
import { AuthService } from '../../core/services/auth.service';
import { Rule, CreateRulePayload, UpdateRulePayload, RuleSeverity, RuleCondition, NotificationChannel } from '../../core/models/rule-alert.model';
import { Device } from '../../core/models/device.model';
import { DEFAULT_PAGE, DEFAULT_PAGE_LIMIT, PAGE_SIZE_OPTIONS } from '../../core/constants/app.constants';

const FEEDBACK_TIMEOUT_MS = 4000;
const DEFAULT_COOLDOWN_SECONDS = 60;
const DEFAULT_METRIC = 'temperature';
const DEFAULT_CONDITION: RuleCondition = '>';
const DEFAULT_THRESHOLD = 35.0;
const DEFAULT_SEVERITY: RuleSeverity = 'warning';
const DEFAULT_CHANNEL: NotificationChannel = 'in_app';

@Component({
  selector: 'app-rules',
  standalone: false,
  templateUrl: './rules.component.html',
})
export class RulesComponent implements OnInit {
  private readonly ruleService = inject(RuleService);
  private readonly deviceService = inject(DeviceService);
  public readonly authService = inject(AuthService);

  public readonly rules = signal<Rule[]>([]);
  public readonly fleetDevices = signal<Device[]>([]);
  public readonly isLoading = signal<boolean>(false);
  public readonly showModal = signal<boolean>(false);
  public readonly editingRuleId = signal<string | null>(null);
  public readonly actionMessage = signal<string | null>(null);

  public readonly currentPage = signal<number>(DEFAULT_PAGE);
  public readonly pageSize = signal<number>(DEFAULT_PAGE_LIMIT);
  public readonly totalCount = signal<number>(0);
  public readonly totalPages = signal<number>(1);
  public readonly pageSizeOptions = PAGE_SIZE_OPTIONS;

  // Form Model
  public formName = '';
  public formDescription = '';
  public formDeviceId = '';
  public formMetricName = DEFAULT_METRIC;
  public formCondition: RuleCondition = DEFAULT_CONDITION;
  public formThreshold: number = DEFAULT_THRESHOLD;
  public formSeverity: RuleSeverity = DEFAULT_SEVERITY;
  public formCooldownSeconds: number = DEFAULT_COOLDOWN_SECONDS;
  public formChannel: NotificationChannel = DEFAULT_CHANNEL;
  public formWebhookUrl = '';
  public formEnabled = true;

  public ngOnInit(): void {
    this.loadRules();
    this.loadDevices();
  }

  public loadRules(page = this.currentPage(), limit = this.pageSize()): void {
    this.isLoading.set(true);
    this.ruleService.getPaginatedRules({ page, limit }).subscribe({
      next: (res) => {
        this.rules.set(res?.data ?? []);
        this.totalCount.set(res?.total ?? 0);
        this.currentPage.set(res?.page ?? page);
        this.pageSize.set(res?.limit ?? limit);
        this.totalPages.set(res?.total_pages ?? 1);
        this.isLoading.set(false);
      },
      error: () => this.isLoading.set(false),
    });
  }

  public loadDevices(): void {
    this.deviceService.getDevices({ all: true }).subscribe({
      next: (devs) => this.fleetDevices.set(devs),
      error: (err) => console.warn('Failed loading devices for rule config:', err),
    });
  }

  public openCreateModal(): void {
    this.editingRuleId.set(null);
    this.resetForm();
    this.showModal.set(true);
  }

  public openEditModal(rule: Rule): void {
    this.editingRuleId.set(rule.id);
    this.formName = rule.name;
    this.formDescription = rule.description;
    this.formDeviceId = rule.device_id ?? '';
    this.formMetricName = rule.metric_name;
    this.formCondition = rule.condition;
    this.formThreshold = rule.threshold;
    this.formSeverity = rule.severity;
    this.formCooldownSeconds = rule.cooldown_seconds;
    this.formChannel = rule.notification_channel;
    this.formWebhookUrl = rule.webhook_url;
    this.formEnabled = rule.enabled;
    this.showModal.set(true);
  }

  public closeModal(): void {
    this.showModal.set(false);
    this.editingRuleId.set(null);
  }

  public handleSaveRule(): void {
    if (!this.formName.trim() || !this.formMetricName.trim()) return;

    const editingId = this.editingRuleId();
    if (editingId) {
      this.executeRuleUpdate(editingId);
      return;
    }
    this.executeRuleCreate();
  }

  public handleToggle(rule: Rule): void {
    this.ruleService.toggleRule(rule.id).subscribe({
      next: (res) => {
        rule.enabled = res.enabled;
        this.setActionFeedback(`Rule "${rule.name}" ${res.enabled ? 'enabled' : 'disabled'}.`);
      },
      error: (err) => alert(err?.error?.error ?? 'Failed to toggle rule.'),
    });
  }

  public handleDelete(rule: Rule): void {
    if (!confirm(`Are you sure you want to delete rule "${rule.name}"?`)) return;

    this.ruleService.deleteRule(rule.id).subscribe({
      next: () => {
        this.loadRules();
        this.setActionFeedback('Rule deleted successfully.');
      },
      error: (err) => alert(err?.error?.error ?? 'Failed to delete rule.'),
    });
  }

  public onPageChange(newPage: number): void {
    this.loadRules(newPage, this.pageSize());
  }

  public onPageSizeChange(newSize: number): void {
    this.pageSize.set(newSize);
    this.loadRules(DEFAULT_PAGE, newSize);
  }

  public getSeverityBadgeClass(severity: RuleSeverity): string {
    switch (severity) {
      case 'critical':
        return 'bg-red-500/10 text-red-500 border-red-500/30';
      case 'warning':
        return 'bg-amber-500/10 text-amber-500 border-amber-500/30';
      default:
        return 'bg-cyan-500/10 text-cyan-500 border-cyan-500/30';
    }
  }

  private executeRuleCreate(): void {
    const payload: CreateRulePayload = {
      name: this.formName.trim(),
      description: this.formDescription.trim(),
      device_id: this.formDeviceId || null,
      metric_name: this.formMetricName.trim(),
      condition: this.formCondition,
      threshold: Number(this.formThreshold),
      severity: this.formSeverity,
      cooldown_seconds: Number(this.formCooldownSeconds),
      notification_channel: this.formChannel,
      webhook_url: this.formWebhookUrl.trim(),
      enabled: this.formEnabled,
    };

    this.ruleService.createRule(payload).subscribe({
      next: () => {
        this.closeModal();
        this.loadRules();
        this.setActionFeedback('Rule created successfully.');
      },
      error: (err) => alert(err?.error?.error ?? 'Failed to create rule.'),
    });
  }

  private executeRuleUpdate(id: string): void {
    const payload: UpdateRulePayload = {
      name: this.formName.trim(),
      description: this.formDescription.trim(),
      device_id: this.formDeviceId || null,
      clear_device_id: !this.formDeviceId,
      metric_name: this.formMetricName.trim(),
      condition: this.formCondition,
      threshold: Number(this.formThreshold),
      severity: this.formSeverity,
      cooldown_seconds: Number(this.formCooldownSeconds),
      notification_channel: this.formChannel,
      webhook_url: this.formWebhookUrl.trim(),
      enabled: this.formEnabled,
    };

    this.ruleService.updateRule(id, payload).subscribe({
      next: () => {
        this.closeModal();
        this.loadRules();
        this.setActionFeedback('Rule updated successfully.');
      },
      error: (err) => alert(err?.error?.error ?? 'Failed to update rule.'),
    });
  }

  private resetForm(): void {
    this.formName = '';
    this.formDescription = '';
    this.formDeviceId = '';
    this.formMetricName = DEFAULT_METRIC;
    this.formCondition = DEFAULT_CONDITION;
    this.formThreshold = DEFAULT_THRESHOLD;
    this.formSeverity = DEFAULT_SEVERITY;
    this.formCooldownSeconds = DEFAULT_COOLDOWN_SECONDS;
    this.formChannel = DEFAULT_CHANNEL;
    this.formWebhookUrl = '';
    this.formEnabled = true;
  }

  private setActionFeedback(msg: string): void {
    this.actionMessage.set(msg);
    setTimeout(() => this.actionMessage.set(null), FEEDBACK_TIMEOUT_MS);
  }
}
