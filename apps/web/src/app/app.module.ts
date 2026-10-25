import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { provideHttpClient, withInterceptorsFromDi, HTTP_INTERCEPTORS } from '@angular/common/http';
import { MatIconModule } from '@angular/material/icon';
import { QueryClient, provideAngularQuery } from '@tanstack/angular-query-experimental';

import { routes } from './app.routes';
import { AppComponent } from './app.component';
import { LoginComponent } from './pages/login/login.component';
import { DeviceListComponent } from './pages/devices/device-list.component';
import { DeviceDetailComponent } from './pages/device-detail/device-detail.component';
import { DashboardComponent } from './pages/dashboard/dashboard.component';
import { AlertsComponent } from './pages/alerts/alerts.component';
import { RulesComponent } from './pages/rules/rules.component';
import { AdminDevicesComponent } from './pages/admin/admin-devices/admin-devices.component';
import { AdminUsersComponent } from './pages/admin/admin-users/admin-users.component';
import { ProfileComponent } from './pages/profile/profile.component';
import { WidgetLibraryComponent } from './pages/widget-library/widget-library.component';
import { WidgetEditorComponent } from './pages/widget-editor/widget-editor.component';
import { TelemetryLiveWidgetComponent } from './widgets/telemetry-live-widget/telemetry-live-widget.component';
import { TelemetryChartWidgetComponent } from './widgets/telemetry-chart-widget/telemetry-chart-widget.component';
import { TelemetryGaugeWidgetComponent } from './widgets/telemetry-gauge-widget/telemetry-gauge-widget.component';
import { TelemetryDigitalCardWidgetComponent } from './widgets/telemetry-digital-card-widget/telemetry-digital-card-widget.component';
import { TelemetrySwitchWidgetComponent } from './widgets/telemetry-switch-widget/telemetry-switch-widget.component';
import { CustomWidgetRunnerComponent } from './widgets/custom-widget-runner/custom-widget-runner.component';
import { MonacoEditorComponent } from './shared/components/monaco-editor/monaco-editor.component';
import { PaginationComponent } from './shared/components/pagination/pagination.component';
import { MomentDatePipe } from './shared/pipes/moment-date.pipe';
import { AuthInterceptor } from './core/interceptors/auth.interceptor';

@NgModule({
  declarations: [
    AppComponent,
    LoginComponent,
    DeviceListComponent,
    DeviceDetailComponent,
    DashboardComponent,
    WidgetLibraryComponent,
    WidgetEditorComponent,
    AlertsComponent,
    RulesComponent,
    AdminDevicesComponent,
    AdminUsersComponent,
    ProfileComponent,
    TelemetryLiveWidgetComponent,
    TelemetryChartWidgetComponent,
    TelemetryGaugeWidgetComponent,
    TelemetryDigitalCardWidgetComponent,
    TelemetrySwitchWidgetComponent,
    CustomWidgetRunnerComponent,
    MonacoEditorComponent,
    PaginationComponent,
    MomentDatePipe,
  ],
  imports: [
    BrowserModule,
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    RouterModule.forRoot(routes),
    MatIconModule,
  ],
  providers: [
    provideAngularQuery(new QueryClient()),
    provideHttpClient(withInterceptorsFromDi()),
    {
      provide: HTTP_INTERCEPTORS,
      useClass: AuthInterceptor,
      multi: true,
    },
  ],
  bootstrap: [AppComponent],
})
export class AppModule {}
