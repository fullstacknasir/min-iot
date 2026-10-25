import { Component, OnInit, OnDestroy, inject, signal } from '@angular/core';
import { Subscription } from 'rxjs';
import { AlertService } from '../../core/services/alert.service';
import { NotificationService } from '../../core/services/notification.service';
import { AuthService } from '../../core/services/auth.service';
import { Alert, RuleSeverity, AlertStatus } from '../../core/models/rule-alert.model';
import { DEFAULT_PAGE, DEFAULT_PAGE_LIMIT, PAGE_SIZE_OPTIONS } from '../../core/constants/app.constants';

const FEEDBACK_TIMEOUT_MS = 4000;
const SEVERITY_CRITICAL = 'critical';
const SEVERITY_WARNING = 'warning';
const STATUS_ACTIVE = 'active';
const STATUS_ACKNOWLEDGED = 'acknowledged';
const STATUS_RESOLVED = 'resolved';

@Component({
  selector: 'app-alerts',
  standalone: false,
  templateUrl: './alerts.component.html',
})
export class AlertsComponent implements OnInit, OnDestroy {
  private readonly alertService = inject(AlertService);
  private readonly notificationService = inject(NotificationService);
  public readonly authService = inject(AuthService);

  public readonly alerts = signal<Alert[]>([]);
  public readonly isLoading = signal<boolean>(false);
  public readonly filterStatus = signal<string>('all');
  public readonly filterSeverity = signal<string>('all');
  public readonly actionMessage = signal<string | null>(null);

  public readonly currentPage = signal<number>(DEFAULT_PAGE);
  public readonly pageSize = signal<number>(DEFAULT_PAGE_LIMIT);
  public readonly totalCount = signal<number>(0);
  public readonly totalPages = signal<number>(1);
  public readonly pageSizeOptions = PAGE_SIZE_OPTIONS;

  private sseSub?: Subscription;

  public ngOnInit(): void {
    this.loadAlerts();
    this.initLiveListener();
  }

  public ngOnDestroy(): void {
    this.sseSub?.unsubscribe();
  }

  public loadAlerts(page = this.currentPage(), limit = this.pageSize()): void {
    this.isLoading.set(true);
    const filter = {
      page,
      limit,
      status: this.filterStatus(),
      severity: this.filterSeverity(),
    };

    this.alertService.getPaginatedAlerts(filter).subscribe({
      next: (res) => {
        this.alerts.set(res?.data ?? []);
        this.totalCount.set(res?.total ?? 0);
        this.currentPage.set(res?.page ?? page);
        this.pageSize.set(res?.limit ?? limit);
        this.totalPages.set(res?.total_pages ?? 1);
        this.isLoading.set(false);
      },
      error: () => this.isLoading.set(false),
    });
  }

  public setFilterStatus(status: string): void {
    this.filterStatus.set(status);
    this.loadAlerts(DEFAULT_PAGE, this.pageSize());
  }

  public setFilterSeverity(severity: string): void {
    this.filterSeverity.set(severity);
    this.loadAlerts(DEFAULT_PAGE, this.pageSize());
  }

  public onPageChange(newPage: number): void {
    this.loadAlerts(newPage, this.pageSize());
  }

  public onPageSizeChange(newSize: number): void {
    this.pageSize.set(newSize);
    this.loadAlerts(DEFAULT_PAGE, newSize);
  }

  public handleAcknowledge(alert: Alert): void {
    this.alertService.acknowledgeAlert(alert.id).subscribe({
      next: () => {
        alert.status = 'acknowledged';
        this.loadAlerts();
        this.setActionFeedback('Alert acknowledged.');
      },
      error: (err) => alertUserError(err?.error?.error ?? 'Failed to acknowledge alert.'),
    });
  }

  public handleResolve(alert: Alert): void {
    this.alertService.resolveAlert(alert.id).subscribe({
      next: () => {
        alert.status = 'resolved';
        this.loadAlerts();
        this.setActionFeedback('Alert resolved successfully.');
      },
      error: (err) => alertUserError(err?.error?.error ?? 'Failed to resolve alert.'),
    });
  }

  public getSeverityBadgeClass(severity: RuleSeverity): string {
    switch (severity) {
      case SEVERITY_CRITICAL:
        return 'bg-red-500/10 text-red-500 border-red-500/30';
      case SEVERITY_WARNING:
        return 'bg-amber-500/10 text-amber-500 border-amber-500/30';
      default:
        return 'bg-cyan-500/10 text-cyan-500 border-cyan-500/30';
    }
  }

  public getStatusBadgeClass(status: AlertStatus): string {
    switch (status) {
      case STATUS_ACTIVE:
        return 'bg-red-500/10 text-red-500 border-red-500/30 animate-pulse';
      case STATUS_ACKNOWLEDGED:
        return 'bg-amber-500/10 text-amber-500 border-amber-500/30';
      case STATUS_RESOLVED:
        return 'bg-emerald-500/10 text-emerald-500 border-emerald-500/30';
      default:
        return 'bg-slate-700/50 text-slate-400 border-slate-600/30';
    }
  }

  private initLiveListener(): void {
    this.sseSub = this.notificationService.subscribeLiveNotifications().subscribe({
      next: () => this.loadAlerts(),
    });
  }

  private setActionFeedback(msg: string): void {
    this.actionMessage.set(msg);
    setTimeout(() => this.actionMessage.set(null), FEEDBACK_TIMEOUT_MS);
  }
}

function alertUserError(msg: string): void {
  alert(msg);
}
