import { Component, OnInit, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import moment from 'moment';
import { CustomWidgetService } from '../../core/services/custom-widget.service';
import { AuthService } from '../../core/services/auth.service';
import { CustomWidget } from '../../core/models/custom-widget.model';
import { WIDGET_PRESETS, WidgetPreset } from '../../core/constants/widget-presets.constants';

const FEEDBACK_TIMEOUT_MS = 3500;

@Component({
  selector: 'app-widget-library',
  standalone: false,
  templateUrl: './widget-library.component.html',
})
export class WidgetLibraryComponent implements OnInit {
  private readonly router = inject(Router);
  private readonly customWidgetService = inject(CustomWidgetService);
  public readonly authService = inject(AuthService);

  public readonly widgets = signal<CustomWidget[]>([]);
  public readonly isLoading = signal<boolean>(false);
  public readonly actionMessage = signal<string | null>(null);
  public readonly presets: WidgetPreset[] = WIDGET_PRESETS;

  public ngOnInit(): void {
    this.loadCustomWidgets();
  }

  public loadCustomWidgets(): void {
    this.isLoading.set(true);
    this.customWidgetService.getCustomWidgets().subscribe({
      next: (list) => {
        this.widgets.set(list);
        this.isLoading.set(false);
      },
      error: () => this.isLoading.set(false),
    });
  }

  public createNewWidget(): void {
    this.router.navigate(['/widgets/editor']);
  }

  public editWidget(id: string): void {
    this.router.navigate(['/widgets/editor', id]);
  }

  public cloneWidget(w: CustomWidget, event?: MouseEvent): void {
    event?.stopPropagation();
    this.customWidgetService
      .createCustomWidget({
        name: `${w.name} (Copy)`,
        description: w.description,
        widget_type: w.widget_type,
        template_html: w.template_html,
        style_css: w.style_css,
        controller_js: w.controller_js,
        settings_schema: w.settings_schema,
      })
      .subscribe({
        next: (created) => {
          this.widgets.update((list) => [created, ...list]);
          this.setActionFeedback(`Widget cloned: "${created.name}"`);
        },
        error: (err) => alert(err?.error?.error ?? 'Failed cloning widget.'),
      });
  }

  public deleteWidget(w: CustomWidget, event?: MouseEvent): void {
    event?.stopPropagation();
    if (!confirm(`Permanently delete custom widget "${w.name}"?`)) return;

    this.customWidgetService.deleteCustomWidget(w.id).subscribe({
      next: () => {
        this.widgets.update((list) => list.filter((item) => item.id !== w.id));
        this.setActionFeedback('Custom widget removed successfully.');
      },
      error: (err) => alert(err?.error?.error ?? 'Failed deleting custom widget.'),
    });
  }

  public formatRelativeTime(dateStr?: string): string {
    return dateStr ? moment(dateStr).fromNow() : 'Just now';
  }

  private setActionFeedback(msg: string): void {
    this.actionMessage.set(msg);
    setTimeout(() => this.actionMessage.set(null), FEEDBACK_TIMEOUT_MS);
  }
}
