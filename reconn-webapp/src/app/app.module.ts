import { HttpClientModule } from '@angular/common/http';
import { NgModule } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { AudioRecorderModule } from './audio_recorder.module';
import { ChatComponent } from './chat.component';
import { ChatModule, ReadbackService, } from './chat.module';
import { CloneVoiceComponent } from './clone_voice.component';
import { TranscribeComponent } from './transcribe.component';
import { TTSComponent } from './tts.component';
import { AIPersonManagementComponent } from './ai-person-mgmt.component';
import { UserManagementComponent } from './user-mgmt.component';

@NgModule({
  declarations: [
    AppComponent,
    CloneVoiceComponent,
    TTSComponent,
    ChatComponent,
    TranscribeComponent,
    AIPersonManagementComponent,
    UserManagementComponent,
  ],
  imports: [
    BrowserModule,
    HttpClientModule,
    AppRoutingModule,
    FormsModule,
    AudioRecorderModule,
    ChatModule,
  ],
  providers: [
    AudioRecorderModule,
    ChatModule,
  ],
  bootstrap: [AppComponent]
})
export class AppModule {}
