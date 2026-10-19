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
import { AdminDevicesComponent } from './pages/admin/admin-devices/admin-devices.component';
import { AdminUsersComponent } from './pages/admin/admin-users/admin-users.component';
import { TelemetryLiveWidgetComponent } from './widgets/telemetry-live-widget/telemetry-live-widget.component';
import { TelemetryChartWidgetComponent } from './widgets/telemetry-chart-widget/telemetry-chart-widget.component';
import { MomentDatePipe } from './shared/pipes/moment-date.pipe';
import { AuthInterceptor } from './core/interceptors/auth.interceptor';

@NgModule({
  declarations: [
    AppComponent,
    LoginComponent,
    DeviceListComponent,
    DeviceDetailComponent,
    AdminDevicesComponent,
    AdminUsersComponent,
    TelemetryLiveWidgetComponent,
    TelemetryChartWidgetComponent,
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
