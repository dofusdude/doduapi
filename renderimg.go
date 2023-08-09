package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type ImageContainerResult struct {
	ContainerId string
	Resolution  string
	AnkamaId    string
}

func RenderVectorImagesWorker(swfFiles []fs.DirEntry, ctx context.Context, resolution string, cli *client.Client, imgOutSubdirName string, done chan bool, result chan []string, yield chan string) {
	var containerIds []string

	for _, swfFile := range swfFiles {
		select {
		case <-done:
			return
		default:
			if swfFile.IsDir() || !strings.HasSuffix(swfFile.Name(), ".swf") {
				continue
			}

			absSwfPath := fmt.Sprintf("%s/data/vector/%s/%s", DockerMountDataPath, imgOutSubdirName, swfFile.Name())
			rawFileName := strings.TrimSuffix(swfFile.Name(), ".swf")
			finalImagePath := fmt.Sprintf("%s/data/img/%s/%s-%s.png", DockerMountDataPath, imgOutSubdirName, rawFileName, resolution)

			if _, err := os.Stat(finalImagePath); err == nil {
				yield <- finalImagePath
				continue
			}

			cmd := []string{"--screenshot", "last", "--screenshot-file", fmt.Sprintf("%s-%s.png", rawFileName, resolution), "-1", "-r1", "--width", resolution, "--height", resolution, swfFile.Name()}
			resp, err := cli.ContainerCreate(ctx, &container.Config{
				Image:      "stelzo/swf-renderer",
				Cmd:        cmd,
				Entrypoint: []string{"/usr/local/bin/dump-gnash"},
				Volumes: map[string]struct{}{
					"/home/developer": {},
				},
			}, &container.HostConfig{
				Binds:      []string{fmt.Sprintf("%s:/home/developer", fmt.Sprintf("%s/data/vector/%s", DockerMountDataPath, imgOutSubdirName))},
				AutoRemove: true,
			}, nil, nil, "")
			if err != nil {
				log.Fatal(err)
			}

			if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
				log.Fatal(err)
			}

			statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
			select {
			case err := <-errCh:
				if err != nil && !strings.Contains(err.Error(), "Error response from daemon: No such container:") {
					log.Fatal(err)
				}
			case <-statusCh:
			}

			srcImagePath := fmt.Sprintf("%s/data/vector/%s/%s-%s.png", DockerMountDataPath, imgOutSubdirName, rawFileName, resolution)
			err = os.Rename(srcImagePath, finalImagePath)
			if err != nil {
				log.Error(err)
				result <- containerIds
				return
			}

			yield <- finalImagePath
			containerIds = append(containerIds, absSwfPath)
		}
	}

	result <- containerIds
}

func RenderVectorImages(done chan bool, imgOutSubdirName string) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	outPath := fmt.Sprintf("%s/data/vector/%s", DockerMountDataPath, imgOutSubdirName)
	ctx := context.Background()
	reader, err := cli.ImagePull(ctx, "stelzo/swf-renderer", types.ImagePullOptions{})
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		log.Fatal(err)
	}

	swfFiles, err := os.ReadDir(outPath)
	if err != nil {
		log.Fatal(err)
	}

	stopChan1 := make(chan bool)
	stopChan2 := make(chan bool)
	stopChan3 := make(chan bool)

	yieldChan1 := make(chan string)
	yieldChan2 := make(chan string)
	yieldChan3 := make(chan string)

	resChan1 := make(chan []string)
	resChan2 := make(chan []string)
	resChan3 := make(chan []string)

	go RenderVectorImagesWorker(swfFiles, ctx, ImgResolutions[0], cli, imgOutSubdirName, stopChan1, resChan1, yieldChan1)
	go RenderVectorImagesWorker(swfFiles, ctx, ImgResolutions[1], cli, imgOutSubdirName, stopChan2, resChan2, yieldChan2)
	go RenderVectorImagesWorker(swfFiles, ctx, ImgResolutions[2], cli, imgOutSubdirName, stopChan3, resChan3, yieldChan3)

	worker1Done := false
	worker2Done := false
	worker3Done := false

	var origPaths [][]string

	for !worker1Done || !worker2Done || !worker3Done {
		select {
		case <-done:
			if !worker1Done {
				stopChan1 <- true
			}
			if !worker2Done {
				stopChan2 <- true
			}
			if !worker3Done {
				stopChan3 <- true
			}
			return
		case imgPath := <-yieldChan1:
			ImgWithResExists.Add(imgPath)
		case imgPath := <-yieldChan2:
			ImgWithResExists.Add(imgPath)
		case imgPath := <-yieldChan3:
			ImgWithResExists.Add(imgPath)
		case res1 := <-resChan1:
			origPaths = append(origPaths, res1)
			worker1Done = true
		case res2 := <-resChan2:
			origPaths = append(origPaths, res2)
			worker2Done = true
		case res3 := <-resChan3:
			origPaths = append(origPaths, res3)
			worker3Done = true
		}
	}

	for _, subPaths := range origPaths {
		for _, path := range subPaths {
			_ = os.Remove(path)
		}
	}

	select {
	case <-done:
		return
	default:
		done <- true
	}
}
