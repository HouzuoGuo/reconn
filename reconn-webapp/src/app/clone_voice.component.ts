import { HttpErrorResponse } from '@angular/common/http';
import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { Observable, ReplaySubject, combineLatest, fromEvent, of, shareReplay } from 'rxjs';
import { catchError, exhaustMap, map, startWith, switchMap, tap } from 'rxjs/operators';
import { AudioRecorderService, ErrorCase, OutputFormat } from './audio_recorder.module';
import { ReadbackResponse, ReadbackService } from './readback.service';
import { CloneRealtimeResponse, VoiceModelResponse, VoiceService } from './voice.service';

@Component({
  selector: 'clone-voice',
  templateUrl: './clone_voice.component.html',
})
export class CloneVoiceComponent implements OnInit {
  userID = '';
  recordingInProgress = false;
  recordButtonCaption = 'Start recording';
  recorderError: Observable<ErrorCase>;
  recording = new ReplaySubject<Blob>(1);

  cloneMessage = new ReplaySubject<string>(1);

  constructor(readonly recorderService: AudioRecorderService, private voiceService: VoiceService) {
    this.recorderError = recorderService.recorderError.pipe(shareReplay({ bufferSize: 1, refCount: true }));
    this.recording.pipe(map((blob: Blob) => {
      console.log('recording blob', blob);
      return "Recording size: " + blob.size / 1024 + "KB";
    })).subscribe(this.cloneMessage);
  }

  ngOnInit() {
  }

  recordButtonClick() {
    if (this.recordingInProgress) {
      this.recorderService.stopRecording(OutputFormat.BLOB).then((blob) => this.recording.next(blob as Blob));
      this.recordingInProgress = false;
      this.recordButtonCaption = 'Start over';
    } else {
      this.recorderService.startRecording();
      this.recordingInProgress = true;
      this.recordButtonCaption = 'Finish recording';
    }
  }

  cloneButtonClick() {
    this.recording.pipe(
      tap(_ => {
        this.cloneMessage.next("Stand by, cloning is in progress.");
      }),
      switchMap((recording: Blob) => {
        console.log('clone input recording', recording);
        return this.voiceService.realTimeClone(this.userID, recording);
      }),
      map((cloneResp: CloneRealtimeResponse) => {
        console.log('clone response', cloneResp);
        return 'Cloned to model file: ' + cloneResp.model;
      }),
      catchError((err: HttpErrorResponse) => {
        console.log('http error', err);
        return of('Error: ' + err.message);
      })
    ).subscribe(this.cloneMessage);
  }
}
