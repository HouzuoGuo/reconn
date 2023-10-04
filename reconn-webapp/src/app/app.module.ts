import { HttpClientModule } from '@angular/common/http';
import { NgModule } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { AudioRecorderModule } from './audio_recorder_module';
import { CloneVoiceComponent } from './clone_voice.component';
import { ReadbackService } from './readback.service';
import { TTSComponent } from './tts.component';
import { VoiceService } from './voice.service';

@NgModule({
  declarations: [
    AppComponent,
    CloneVoiceComponent,
    TTSComponent,
  ],
  imports: [
    BrowserModule,
    HttpClientModule,
    AppRoutingModule,
    FormsModule,
    AudioRecorderModule,
  ],
  providers: [
    ReadbackService,
    VoiceService,
    AudioRecorderModule,
  ],
  bootstrap: [AppComponent]
})
export class AppModule {}
